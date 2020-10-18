require 'provider/abstract_core'

module Provider
  class TFLint < Provider::Terraform
    def generate_resource(pwd, data, generate_code, generate_docs)
      @rule_names = []

      tf_product = (@config.legacy_name || data.product.name).underscore
      @terraform_name = data.object.legacy_name || "google_#{tf_product}_#{data.object.name.underscore}"

      data.object.all_user_properties.each do |prop|
        next if prop.output
        next unless prop.is_a?(Api::Type::Enum) || prop.validation&.regex

        @prop = prop
        @rule_name = "#{@terraform_name}_invalid_#{prop.name.underscore}"

        data.generate(pwd, '/templates/tflint/rule.go.erb', "#{@rule_name}.go", self)

        @rule_names << @rule_name
      end

      return if @rule_names.empty?
      # FIXME: Keep rule names independent of external files
      File.open('/tmp/rulenames.txt', 'a') do |f|
        @rule_names.each do |name|
          f.puts "#{name}\n"
        end
      end
    end

    def generate_resource_tests(pwd, data) end

    def generate_resource_sweepers(pwd, data) end

    def generate_iam_policy(pwd, data, generate_code, generate_docs) end

    def generate_operation(pwd, output_folder, _types) end

    def compile_common_files(output_folder, products, common_compile_file)
      # FIXME: Keep rule names independent of external files
      @rule_names = File.read('/tmp/rulenames.txt').split("\n").sort
      File.delete('/tmp/rulenames.txt')

      Google::LOGGER.info 'Compiling common files.'
      file_template = ProviderFileTemplate.new(
        output_folder,
        @target_version_name,
        build_env,
        products
      )
      compile_file_list(output_folder, [['provider.go', 'templates/tflint/provider.go.erb']], file_template)
    end

    def copy_common_files(output_folder, generate_code, generate_docs)
      Google::LOGGER.info 'Copying common files.'
      copy_file_list(output_folder, [['validation.go', 'third_party/terraform/utils/validation.go']])
    end
  end
end
