// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-present Datadog, Inc.

//go:build kubeapiserver
// +build kubeapiserver

package mutate

import (
	"errors"
	"fmt"
	"strings"

	"github.com/DataDog/datadog-agent/pkg/config"
	"github.com/DataDog/datadog-agent/pkg/util/log"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/dynamic"
)

const (
	volumeName           = "datadog-auto-instrumentation"
	mountPath            = "/datadog"
	javaToolOptionsKey   = "JAVA_TOOL_OPTIONS"
	javaToolOptionsValue = " -javaagent:/datadog/dd-java-agent.jar"
)

var (
	tracerVersionLabelKeyFormat     = "admission.datadoghq.com/%s-tracer.version"
	customTracerAnnotationKeyFormat = "admission.datadoghq.com/%s-tracer.custom-image"
	supportedLanguages              = []string{
		"java",
		"python",
		"node",
	}
)

// InjectAutoInstru injects APM libraries into pods
func InjectAutoInstru(rawPod []byte, ns string, dc dynamic.Interface) ([]byte, error) {
	return mutate(rawPod, ns, injectAutoInstru, dc)
}

func injectAutoInstru(pod *corev1.Pod, _ string, _ dynamic.Interface) error {
	if pod == nil {
		return errors.New("cannot inject lib into nil pod")
	}

	language, image, shouldInject := extractLibInfo(pod, config.Datadog.GetString("admission_controller.auto_instru.container_registry"))
	if !shouldInject {
		return nil
	}

	log.Infof("Injecting image %s", image)

	return injectAutoInstruConfig(pod, language, image)
}

func extractLibInfo(pod *corev1.Pod, containerRegistry string) (string, string, bool) {
	podAnnotations := pod.GetAnnotations()
	podLabels := pod.GetLabels()
	for _, lang := range supportedLanguages {
		if image, found := podAnnotations[fmt.Sprintf(customTracerAnnotationKeyFormat, lang)]; found {
			return lang, image, true
		}

		if version, found := podLabels[fmt.Sprintf(tracerVersionLabelKeyFormat, lang)]; found {
			image := fmt.Sprintf("%s/%s:%s", containerRegistry, "apm-"+lang, version) // TODO: update the repo name (temporarily using apm-<lang>)
			return lang, image, true
		}
	}

	return "", "", false
}

func injectAutoInstruConfig(pod *corev1.Pod, language, image string) error {
	switch strings.ToLower(language) {
	case "java":
		err := injectJavaInitContainer(pod, image)
		if err != nil {
			return err
		}

		err = injectJavaConfig(pod)
		if err != nil {
			return err
		}

	case "python", "node":
		// TODO
		return fmt.Errorf("language %q is not implemented yet", language)
	default:
		return fmt.Errorf("language %q is not supported", language)
	}

	return nil
}

func injectJavaInitContainer(pod *corev1.Pod, image string) error {
	containerName := "datadog-tracer-init"
	podStr := podString(pod)
	log.Debugf("Injecting init container named %q with image %q into pod %s", containerName, image, podStr)
	for _, container := range pod.Spec.InitContainers {
		if container.Name == containerName {
			return fmt.Errorf("init container %q already exists in pod %q", containerName, podStr)
		}
	}

	pod.Spec.InitContainers = append(pod.Spec.InitContainers,
		corev1.Container{
			Name:    containerName,
			Image:   image,
			Command: []string{"sh", "copy-javaagent.sh", mountPath},
			VolumeMounts: []corev1.VolumeMount{
				{
					Name:      volumeName,
					MountPath: mountPath,
				},
			},
		})

	return nil
}

func injectJavaConfig(pod *corev1.Pod) error {
	pod.Spec.Volumes = append(pod.Spec.Volumes, corev1.Volume{
		Name: volumeName,
		VolumeSource: corev1.VolumeSource{
			EmptyDir: &corev1.EmptyDirVolumeSource{},
		},
	})

	for i, ctr := range pod.Spec.Containers {
		index := envIndex(ctr.Env, javaToolOptionsKey)
		if index < 0 {
			pod.Spec.Containers[i].Env = append(pod.Spec.Containers[i].Env, corev1.EnvVar{
				Name:  javaToolOptionsKey,
				Value: javaToolOptionsValue,
			})
		} else {
			if pod.Spec.Containers[i].Env[index].ValueFrom != nil {
				return errors.New("JAVA_TOOL_OPTIONS is defined via ValueFrom")
			}

			pod.Spec.Containers[i].Env[index].Value = pod.Spec.Containers[i].Env[index].Value + javaToolOptionsValue
		}

		pod.Spec.Containers[i].VolumeMounts = append(pod.Spec.Containers[i].VolumeMounts, corev1.VolumeMount{Name: volumeName, MountPath: mountPath})
	}

	return nil
}
