package ai.open.right.workflow.flow.llm.provider.google;

import ai.open.right.utils.HostUtils;
import ai.open.right.utils.IPUtils;
import ai.open.right.utils.JsonUtils;
import jakarta.annotation.PostConstruct;
import lombok.Getter;
import lombok.Setter;
import lombok.extern.slf4j.Slf4j;
import org.apache.commons.io.IOUtils;
import org.apache.commons.lang3.StringUtils;
import org.apache.http.HttpResponse;
import org.apache.http.HttpStatus;
import org.apache.http.client.methods.HttpGet;
import org.apache.http.client.methods.HttpRequestBase;
import org.apache.http.impl.nio.client.CloseableHttpAsyncClient;
import org.springframework.beans.BeanUtils;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.beans.factory.annotation.Qualifier;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.boot.autoconfigure.condition.ConditionalOnMissingBean;
import org.springframework.boot.autoconfigure.condition.ConditionalOnProperty;
import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Configuration;
import org.springframework.util.Assert;

import java.io.InputStream;
import java.nio.charset.StandardCharsets;
import java.util.Map;

@Slf4j
@Setter
@Getter
public class VertexTokenExchange {

    public static final String NAME = "VertexTokenExchange";

    public static final String LOCATION = "#app_id";

    public static final String CLIENT = "#client";

    public static final String IP = "#ip";

    protected CloseableHttpAsyncClient other;

    // Vertex Project
    protected String project;

    // Vertex Remote Token Service URL
    protected String remote;

    // Vertex APP ID
    protected String appId;

    protected String url;

    @PostConstruct
    public void init() throws Exception {
        if (!StringUtils.isEmpty(this.project) || !StringUtils.isEmpty(this.appId)) {
            // 精确匹配
            this.url = this.remote.replace(VertexTokenExchange.LOCATION, StringUtils.defaultString(this.appId, this.project)).replace(VertexTokenExchange.CLIENT, StringUtils.defaultString(HostUtils.getHostName(), "")).replace(VertexTokenExchange.IP, StringUtils.defaultString(IPUtils.getIP(), ""));
            if (log.isInfoEnabled()) {
                log.info("Vertex remote token service url={}", this.url);
            }
        }
    }

    public String exchange() throws Exception {
        if (StringUtils.isEmpty(this.url)) {
            if (log.isDebugEnabled()) {
                log.debug("Vertex remote token service url can not be empty");
            }
            return null;
        }
        HttpResponse response = this.response(new HttpGet(this.url));
        Assert.isTrue(response.getStatusLine().getStatusCode() == HttpStatus.SC_OK, "Vertex token request has failed: " + response.getStatusLine().getStatusCode());
        try (InputStream input = response.getEntity().getContent()) {
            Map<String, String> data = JsonUtils.read(IOUtils.toString(input, StandardCharsets.UTF_8), Map.class);
            String accessToken = String.class.cast(data.get("data"));
            Assert.hasText(accessToken, "Vertex token can not be empty");
            return accessToken;
        }
    }

    protected HttpResponse response(HttpRequestBase httpRequestBase) throws Exception {
        return this.other.execute(httpRequestBase, null).get();
    }

    @ConditionalOnProperty(name = "vertex.enable", havingValue = "true", matchIfMissing = false)
    @Configuration
    @Setter
    @Getter
    public static class InitConfig {

        @Autowired
        @Qualifier("other")
        protected CloseableHttpAsyncClient other;

        @Value("${spring.application.name:}")
        // Vertex Project
        protected String project;

        @Value("${vertex.token.exchange.remote:}")
        // Vertex Remote Token Service URL
        protected String remote;

        @Value("${vertex.token.exchange.app.id:}")
        // Vertex APP ID
        protected String appId;

        @Bean(name = VertexTokenExchange.NAME)
        @ConditionalOnMissingBean(name = VertexTokenExchange.NAME)
        public VertexTokenExchange vertexTokenExchange() throws Exception {
            VertexTokenExchange vertexTokenExchange = new VertexTokenExchange();
            BeanUtils.copyProperties(this, vertexTokenExchange);
            log.info("VertexTokenExchange inited. project={},remote={},appId={}", vertexTokenExchange.getProject(), vertexTokenExchange.getRemote(), vertexTokenExchange.getAppId());
            return vertexTokenExchange;
        }
    }
}
