package ai.open.right;

import org.springframework.boot.autoconfigure.AutoConfigureOrder;
import org.springframework.context.annotation.ComponentScan;
import org.springframework.context.annotation.PropertySource;
import org.springframework.context.annotation.PropertySources;
import org.springframework.core.Ordered;
import org.springframework.scheduling.annotation.EnableAsync;
import org.springframework.scheduling.annotation.EnableScheduling;

@PropertySources({
        @PropertySource("classpath:right-global.properties"),
        @PropertySource("classpath:right-thread.properties")
})
@ComponentScan(basePackages = "ai.open.right")
@AutoConfigureOrder(Ordered.LOWEST_PRECEDENCE)
@EnableAsync(proxyTargetClass = true)
@EnableScheduling
public class AutoConfiguration {

}
