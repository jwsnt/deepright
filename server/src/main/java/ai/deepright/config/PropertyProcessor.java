package ai.deepright.config;

import static org.springframework.util.ObjectUtils.isEmpty;

import lombok.extern.slf4j.Slf4j;
import org.springframework.boot.SpringApplication;
import org.springframework.boot.env.EnvironmentPostProcessor;
import org.springframework.core.env.ConfigurableEnvironment;
import org.springframework.core.env.MutablePropertySources;
import org.springframework.core.env.PropertiesPropertySource;
import org.springframework.core.io.ClassPathResource;
import org.springframework.core.io.FileSystemResource;
import org.springframework.core.io.support.PropertiesLoaderUtils;

import java.util.Properties;

@Slf4j
public class PropertyProcessor implements EnvironmentPostProcessor {

    // 外部配置路径的系统属性 key，例如 -Dconfig.path=/etc/myapp/
    public static final String CONFIG_PATH_KEY = "config.path";

    // 需要加载的配置文件列表：先加载 classpath 默认值，再用 -Dconfig.path 指定目录下的同名文件覆盖
    private static final String[] CONFIG_FILES = {
            "application.properties",
            "right-global.properties",
            "right-thread.properties",
    };

    @Override
    public void postProcessEnvironment(ConfigurableEnvironment environment, SpringApplication application) {
        MutablePropertySources propertySources = environment.getPropertySources();
        String configPath = environment.getProperty(PropertyProcessor.CONFIG_PATH_KEY);
        for (String fileName : PropertyProcessor.CONFIG_FILES) {
            Properties props = loadProperties(fileName, configPath);
            if (!props.isEmpty()) {
                // addFirst: 外部加载的优先级高于jar内默认值
                propertySources.addFirst(new PropertiesPropertySource(fileName, props));
                log.info("Loaded [{}] into environment", fileName);
            }
        }
    }

    protected Properties loadProperties(String fileName, String configPath) {
        Properties merged = new Properties();
        // 1) 先加载 classpath 默认文件，作为缺省值
        Properties props = loadFromClasspath(fileName);
        if (props != null) {
            merged.putAll(props);
        }
        // 2) 再加载 config.path 下的同名文件，覆盖默认值
        if (configPath != null && !configPath.isBlank()) {
            String normalized = configPath.endsWith("/") ? configPath : configPath + "/";
            props = loadFromFileSystem(normalized + fileName);
            if (props != null) {
                merged.putAll(props);
            }
        }
        if (!merged.isEmpty()) {
            return merged;
        }
        log.debug("No config found for [{}], skipped", fileName);
        // 返回空 PropertySource，不阻塞启动
        return merged;
    }

    protected Properties loadFromFileSystem(String absolutePath) {
        try {
            FileSystemResource resource = new FileSystemResource(absolutePath);
            if (resource.exists()) {
                log.info("Loading config from file system: {}", absolutePath);
                return PropertiesLoaderUtils.loadProperties(resource);
            }
        } catch (Exception e) {
            log.warn("Failed to load [{}]: {}", absolutePath, e.getMessage());
        }
        return null;
    }

    private Properties loadFromClasspath(String classpathFile) {
        try {
            ClassPathResource resource = new ClassPathResource(classpathFile);
            if (resource.exists()) {
                log.info("Loading config from classpath: {}", classpathFile);
                return PropertiesLoaderUtils.loadProperties(resource);
            }
        } catch (Exception e) {
            log.warn("Failed to load [{}]: {}", classpathFile, e.getMessage());
        }
        return null;
    }
}
