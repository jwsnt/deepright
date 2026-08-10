package ai.open.right.config;

import com.google.common.collect.ImmutableMap;
import lombok.extern.slf4j.Slf4j;
import org.apache.commons.lang3.StringUtils;
import org.springframework.boot.SpringApplication;
import org.springframework.boot.env.EnvironmentPostProcessor;
import org.springframework.boot.system.ApplicationHome;
import org.springframework.core.env.ConfigurableEnvironment;
import org.springframework.core.env.MapPropertySource;
import org.springframework.core.env.PropertySource;

import java.io.File;

/**
 * 环境属性配置处理器
 * 1. 自动计算项目路径并注入 project 属性
 * 2. 增强环境变量读取，支持 . 到 _ 的转换
 */
public class PropertiesConfig implements EnvironmentPostProcessor {

    protected static final String RESOURCE = File.separator + "BOOT-INF" + File.separator + "classes";

    protected static final String SUFFIX = ".jar";

    @Override
    public void postProcessEnvironment(ConfigurableEnvironment environment, SpringApplication application) {
        // 构建并注入项目路径属性
        this.buildProject(environment, application);
        // 替换系统环境变量属性源，以支持更灵活的命名映射
        for (PropertySource<?> source : environment.getPropertySources()) {
            if ("systemEnvironment".equals(source.getName())) {
                environment.getPropertySources().replace(source.getName(), new SysEnvPropertySource(source));
                break;
            }
        }
    }

    protected void buildProject(ConfigurableEnvironment environment, SpringApplication application) {
        // 获取应用主目录
        ApplicationHome home = this.buildApp(application);
        String source = home.getSource() != null ? home.getSource().getAbsolutePath() : ".";
        // 如果是 JAR 运行，截掉 .jar 后缀并拼接 BOOT-INF/classes
        String project = (StringUtils.endsWithIgnoreCase(source, PropertiesConfig.SUFFIX) ? source.substring(0, source.length() - PropertiesConfig.SUFFIX.length()) : source) + PropertiesConfig.RESOURCE;
        // 将 project 属性添加到配置源的最前面
        environment.getPropertySources().addFirst(new MapPropertySource("project", ImmutableMap.of("project", project)));
    }

    protected ApplicationHome buildApp(SpringApplication application) {
        return new ApplicationHome(application.getMainApplicationClass());
    }

    /**
     * 自定义环境变量属性源，支持将 a.b.c 映射为 a_b_c 或 A_B_C
     */
    @Slf4j
    public static class SysEnvPropertySource extends PropertySource<Object> {

        protected final PropertySource<?> delegate;

        public SysEnvPropertySource(PropertySource<?> delegate) {
            super(delegate.getName(), delegate.getSource());
            this.delegate = delegate;
        }

        @Override
        public Object getProperty(String name) {
            // 1. 尝试将 . 替换为 _
            String name_env = name.replace('.', '_');
            Object value = this.delegate.getProperty(name_env);
            // 2. 尝试转换为大写下划线格式
            value = value != null ? value : this.delegate.getProperty(name_env = name_env.toUpperCase());
            if (value != null) {
                if (log.isDebugEnabled()) {
                    log.debug("Loading property from env={}", name_env);
                }
                return value;
            }
            // 3. 回退到原始名称查找
            return this.delegate.getProperty(name);
        }
    }
}