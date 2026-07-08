package ai.deepright.module;

import static org.springframework.util.ObjectUtils.isEmpty;

import ai.open.right.utils.IPUtils;
import ai.open.right.workflow.flow.media.MediaTransferUtils;
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

    public String dataHost(String url) throws Exception {
        return MediaTransferUtils.isNetwork(url) ? url : (this.host() + this.data() + "?name=" + URLEncoder.encode(Paths.get(url).getFileName().toString(), StandardCharsets.UTF_8));
    }

    public String dataHost() throws Exception {
        return this.host() + this.data() + "?name=";
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
