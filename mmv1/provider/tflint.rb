module Provider
  class TFLint < Provider::Terraform
    @@rule_names = []
    @@resource_url = {}

    def generate_resource(pwd, data, generate_code, generate_docs)
      @terraform_name = data.object.legacy_name || "google_#{full_resource_name(data)}"

      begin
        @@resource_url[@terraform_name] = URI.parse(data.product.base_url).host
      rescue URI::InvalidURIError => exn
        Google::LOGGER.warn "Cannot parse Api::Product#base_url: #{exn}"
      end

      data.object.all_user_properties.each do |prop|
        next if prop.output
        next unless prop.is_a?(Api::Type::Enum) || prop.validation&.regex

        @prop = prop
        @rule_name = "#{@terraform_name}_invalid_#{prop.name.underscore}"

        data.generate(pwd, '/templates/tflint/rule.go.erb', "#{@rule_name}.go", self)

        @@rule_names << @rule_name
      end
    end

    def generate_resource_tests(pwd, data) end

    def generate_resource_sweepers(pwd, data) end

    def generate_iam_policy(pwd, data, generate_code, generate_docs) end

    def generate_operation(pwd, output_folder, _types) end

    def compile_common_files(output_folder, products, common_compile_file)
      @rule_names = @@rule_names.sort
      @resource_url = @@resource_url

      Google::LOGGER.info 'Compiling common files.'
      file_template = ProviderFileTemplate.new(
        output_folder,
        @target_version_name,
        build_env,
        products
      )
      compile_file_list(output_folder, [['provider.go', 'templates/tflint/provider.go.erb']], file_template)
      compile_file_list(output_folder, [['api_definition.go', 'templates/tflint/api_definition.go.erb']], file_template)
    end

    def copy_common_files(output_folder, generate_code, generate_docs)
      Google::LOGGER.info 'Copying common files.'
      copy_file_list(output_folder, [
                       ['verify/validation.go',
                        'third_party/terraform/verify/validation.go'],
                    ])
    end
  end
end
