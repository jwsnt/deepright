package ai.deepright.module;

import ai.deepright.feature.FeatureUtils;
import ai.open.right.utils.IPUtils;
import ai.open.right.workflow.flow.WorkflowTask;
import ai.open.right.workflow.flow.media.MediaTransferUtils;
import jakarta.annotation.PostConstruct;
import lombok.Getter;
import lombok.Setter;
import lombok.extern.slf4j.Slf4j;
import org.apache.commons.lang3.StringUtils;
import org.springframework.beans.BeanUtils;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.boot.autoconfigure.condition.ConditionalOnMissingBean;
import org.springframework.boot.autoconfigure.condition.ConditionalOnProperty;
import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Configuration;

import java.net.URI;
import java.net.URLEncoder;
import java.nio.charset.StandardCharsets;
import java.nio.file.Paths;

@Slf4j
@Setter
@Getter
// HTTP 上下行收口
public class HttpProtocol {

    protected String protocol;

    protected Integer port;

    protected String host;

    protected String data;

    @PostConstruct
    public void init() throws Exception {
        if (!StringUtils.isEmpty(this.host)) {
            // 配置完整 Host 时，使用其协议生成基于请求 Host 的下载地址
            String scheme = URI.create(this.host).getScheme();
            if (!StringUtils.isEmpty(scheme)) {
                this.protocol = scheme + "://";
            }
        }
    }

    public String dataHost(WorkflowTask workTask, String url) throws Exception {
        // 先从Http Host头取（格式为host:port，不带协议头）
        return MediaTransferUtils.isNetwork(url) ? url : (this.host(workTask) + this.data() + "?name=" + URLEncoder.encode(Paths.get(url).getFileName().toString(), StandardCharsets.UTF_8));
    }

    public String dataHost() throws Exception {
        return this.host() + this.data() + "?name=";
    }

    public String host(WorkflowTask workTask) throws Exception {
        // metadata.Host 由 HTTP Host header 注入，优先使用请求实际访问的 Host
        String host = FeatureUtils.buildHost(workTask);
        return !StringUtils.isEmpty(host) ? this.protocol + host : this.host();
    }

    public String host() throws Exception {
        return StringUtils.isEmpty(this.host) ? (this.protocol + IPUtils.getIP() + ":" + this.port) : this.host;
    }

    public String data() throws Exception {
        return this.data;
    }

    @ConditionalOnProperty(name = "chat.http.enable", havingValue = "true", matchIfMissing = true)
    @Configuration
    @Setter
    @Getter
    public static class InitConfig {

        @Value("${chat.http.protocol:http://}")
        protected String protocol;

        @Value("${chat.http.port:}")
        protected Integer port;

        @Value("${chat.http.data:/data}")
        protected String data;

        @Value("${chat.http.host:}")
        protected String host;

        @Bean
        @ConditionalOnMissingBean(value = HttpProtocol.class)
        public HttpProtocol httpProtocol() throws Exception {
            HttpProtocol httpProtocol = new HttpProtocol();
            BeanUtils.copyProperties(this, httpProtocol);
            log.info("HttpProtocol inited");
            return httpProtocol;
        }
    }
}
