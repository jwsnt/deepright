package ai.open.right.release;

import ai.open.right.WorkflowException;
import ai.open.right.resouce.ResourceService;
import ai.open.right.utils.JarUtils;
import jakarta.annotation.PostConstruct;
import lombok.Getter;
import lombok.Setter;
import lombok.extern.slf4j.Slf4j;
import org.apache.commons.lang3.StringUtils;
import org.apache.commons.lang3.builder.ToStringBuilder;
import org.springframework.beans.BeanUtils;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.boot.autoconfigure.condition.ConditionalOnMissingBean;
import org.springframework.boot.system.ApplicationHome;
import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Configuration;

import java.io.File;
import java.util.Arrays;
import java.util.HashSet;
import java.util.Set;

@Slf4j
@Setter
@Getter
public class ResourceReleaser {

    public static final String KEY_NESTED = "nested:";

    public static final String KEY_JAR = "jar:";

    public static final String KEY = ResourceReleaser.KEY_JAR + ResourceReleaser.KEY_NESTED;

    protected ResourceService resourceService;

    // 是否释放（解压）Jar内资源
    protected Boolean release = false;

    // 需要解压文件的后缀名
    protected String suffix = "";

    // 需要解压的文件路径（默认为当前Jar）
    protected String file;

    @PostConstruct
    public void init() {
        if (this.release) {
            Set<String> suffix = StringUtils.isEmpty(this.suffix) ? new HashSet<String>() : new HashSet<String>(Arrays.asList(this.suffix.split(",")));
            log.info("The suffix={}", suffix);
            try {
                // 路径均有使用者确认路径安全
                File jar = StringUtils.isEmpty(this.file) ? this.getJar() : new File(this.file);
                if (jar != null && jar.exists() && jar.isFile()) {
                    if (log.isInfoEnabled()) {
                        log.info("The jar={}", jar.getAbsolutePath());
                    }
                    JarUtils.unzip(jar, suffix);
                }
            } catch (Exception e) {
                WorkflowException.dolog(e);
            }
        }
    }

    // /target/xxx.jar
    public String getRoot() throws Exception {
        ApplicationHome home = this.getHome();
        return home.getSource().getAbsolutePath();
    }

    protected ApplicationHome getHome() throws Exception {
        return new ApplicationHome(this.resourceService.root());
    }

    protected File getJar() throws Exception {
        String url = this.getUrl();
        if (url.startsWith(ResourceReleaser.KEY_JAR)) {
            int endIndex = url.indexOf("!");
            if (endIndex > 0) {
                File jar = new File(url.substring(0, endIndex).replaceFirst(ResourceReleaser.KEY, ""));
                if (log.isDebugEnabled()) {
                    log.debug("The file={}", jar);
                }
                return jar;
            }
        }
        return null;
    }

    protected String getUrl() throws Exception {
        // jar:nested:/Your Path/target/xxx.jar/!BOOT-INF/classes/!/
        String url = ResourceReleaser.KEY + this.getRoot() + "/!BOOT-INF/classes/!/";
        if (log.isDebugEnabled()) {
            log.debug("The url={}", url);
        }
        return url;
    }

    @Configuration
    @Setter
    @Getter
    public static class InitConfig {

        @Autowired
        protected ResourceService resourceService;

        @Value("${resource.release:false}")
        // 是否释放（解压）Jar内资源
        protected Boolean release = false;

        @Value("${resource.suffix:}")
        // 需要解压文件的后缀名
        protected String suffix = "";

        @Value("${resource.file:}")
        // 需要解压的文件路径（默认为当前Jar）
        protected String file;

        @Bean
        @ConditionalOnMissingBean(value = ResourceReleaser.class)
        public ResourceReleaser resourceConfig() throws Exception {
            ResourceReleaser resourceConfig = new ResourceReleaser();
            BeanUtils.copyProperties(this, resourceConfig);
            log.info("ResourceConfig inited={}", ToStringBuilder.reflectionToString(resourceConfig));
            return resourceConfig;
        }
    }
}
