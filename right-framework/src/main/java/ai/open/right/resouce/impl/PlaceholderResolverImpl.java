package ai.open.right.resouce.impl;

import ai.open.right.resouce.PlaceholderResolver;
import lombok.Getter;
import lombok.Setter;
import lombok.extern.slf4j.Slf4j;
import org.apache.commons.lang3.StringUtils;
import org.springframework.beans.BeanUtils;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.boot.autoconfigure.condition.ConditionalOnMissingBean;
import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Configuration;
import org.springframework.core.env.Environment;

import java.util.regex.Matcher;
import java.util.regex.Pattern;

@Setter
@Getter
@Slf4j
public class PlaceholderResolverImpl implements PlaceholderResolver {

    private static final Pattern PLACEHOLDER_PATTERN = Pattern.compile("\\$\\{([^}]+)}");

    protected Environment environment;

    protected String prefix;

    @Override
    public String replace(String input) throws Exception {
        if (StringUtils.isEmpty(input)) {
            return input;
        }
        Matcher matcher = PlaceholderResolverImpl.PLACEHOLDER_PATTERN.matcher(input);
        StringBuffer result = new StringBuffer();
        while (matcher.find()) {
            String placeholder = matcher.group(1);
            if (StringUtils.isEmpty(this.prefix) || StringUtils.startsWithIgnoreCase(placeholder, this.prefix)) {
                String key = "${" + placeholder + "}";
                String replacement = this.environment.getProperty(placeholder, key);
                matcher.appendReplacement(result, Matcher.quoteReplacement(replacement));
            }
        }
        matcher.appendTail(result);
        String output = result.toString();
        if (log.isDebugEnabled() && StringUtils.equalsIgnoreCase(input, output)) {
            log.debug("Replace config={}", output);
        }
        return output;
    }
    @Configuration
    @Setter
    @Getter
    public static class InitConfig {

        @Autowired
        protected Environment environment;

        // 安全前缀，推荐配置
        @Value("${placeholder.prefix:}")
        protected String prefix;

        @Bean
        @ConditionalOnMissingBean(value = PlaceholderResolver.class)
        public PlaceholderResolver placeholderResolver() throws Exception {
            PlaceholderResolverImpl placeholderResolver = new PlaceholderResolverImpl();
            BeanUtils.copyProperties(this, placeholderResolver);
            log.info("PlaceholderResolverImpl inited, prefix={} ", placeholderResolver.getPrefix());
            return placeholderResolver;
        }
    }
}