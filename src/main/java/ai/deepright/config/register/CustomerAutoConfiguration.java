package ai.deepright.config.register;

import org.springframework.boot.autoconfigure.AutoConfiguration;
import org.springframework.boot.autoconfigure.AutoConfigureBefore;
import org.springframework.context.annotation.Import;

@AutoConfiguration
@AutoConfigureBefore(ai.open.right.AutoConfiguration.class)
@Import(CustomerConfigurationRegistrar.class)
public class CustomerAutoConfiguration {
}
