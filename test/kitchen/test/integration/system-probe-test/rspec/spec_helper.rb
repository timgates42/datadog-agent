require "rspec/core/formatters/base_text_formatter"

class CustomFormatter
  RSpec::Core::Formatters.register self, :example_passed, :example_failed, :dump_summary, :dump_failures, :example_group_started, :example_group_finished

  def initialize(output)
    @output = output
    @release = `uname -r`.strip
  end

  # Remove "."'s from the test execution output
  def example_passed(_)
  end

  # Remove "F"'s from the test execution output
  def example_failed(_)
  end

  def example_group_started(notification)
    @output << "\n[#{@release}] started #{notification.group.description}\n"
  end

  def example_group_finished(notification)
    @output << "[#{@release}] finished #{notification.group.description}\n\n"
  end

  def dump_summary(notification)
    @output << "[#{@release}] Finished in #{RSpec::Core::Formatters::Helpers.format_duration(notification.duration)}.\n"
    @output << "[#{@release}] Platform: #{`uname -a`}\n\n"
  end

  def dump_failures(notification) # ExamplesNotification
    if notification.failed_examples.length > 0
      failures = RSpec::Core::Formatters::ConsoleCodes.wrap("[#{@release}] FAILURES:", :failure)
      @output << "\n#{failures}\n\n"
      @output << error_summary(notification)
    end
  end

  private

  def error_summary(notification)
    summary_output = notification.failed_examples.map do |example|
      "#{example.full_description}:\n#{example.execution_result.exception.message}\n\n"
    end

    summary_output.join
  end
end


RSpec.configure do |config|
  config.formatter = CustomFormatter
end
