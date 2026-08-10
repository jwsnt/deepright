package ai.open.right.workflow.flow.llm.provider.google;

import ai.open.right.workflow.flow.llm.config.LLMConfig;
import ai.open.right.workflow.flow.llm.provider.ProviderRouter;
import lombok.Getter;
import lombok.Setter;
import lombok.extern.slf4j.Slf4j;
import org.apache.commons.collections.MapUtils;
import org.apache.commons.lang3.StringUtils;
import org.springframework.beans.BeanUtils;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.boot.autoconfigure.condition.ConditionalOnMissingBean;
import org.springframework.boot.autoconfigure.condition.ConditionalOnProperty;
import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Configuration;
import org.springframework.util.Assert;

@Setter
@Getter
@Slf4j
public class VertexRouter extends GoogleRouter {

    public static final String NAME = "VertexRouter";

    public static final String PROJECT_ID = "project_id";

    public static final String LOCATION = "location";

    public static final String MODEL = "model";

    // Vertex Project ID
    protected String projectId;

    // Vertex API Location
    protected String location;

    // Vertex Stream API URL
    protected String urlStream;

    // Vertex非流式API URL
    protected String urlOnce;

    @Override
    protected String url(GoogleRequest request, LLMConfig llmConfig, String t) throws Exception {
        String projectId = String.valueOf(llmConfig.getAdditional().getOrDefault(VertexRouter.PROJECT_ID, this.projectId));
        String location = String.valueOf(llmConfig.getAdditional().getOrDefault(VertexRouter.LOCATION, this.location));
        Assert.hasText(request.getModel(), "Vertex model can not be empty");
        Assert.hasText(projectId, "Vertex project id can not be empty");
        Assert.hasText(location, "Vertex location can not be empty");
        String url = null;
        if (ProviderRouter.URL_STREAM.equals(t)) {
            // 精确匹配
            url = MapUtils.getString(request.getMessage().getMetadata(), "__url", StringUtils.defaultIfEmpty(request.getUrl(), this.urlStream).replace("#projectid", projectId).replace("#location", location).replace("#model", request.getModel()));
        } else {
            url = MapUtils.getString(request.getMessage().getMetadata(), "__url", StringUtils.defaultIfEmpty(request.getUrl(), this.urlOnce).replace("#projectid", projectId).replace("#location", location).replace("#model", request.getModel()));
        }
        Assert.hasText(url, "Url can not be empty");
        return url;
    }

    @ConditionalOnProperty(name = "vertex.enable", havingValue = "true", matchIfMissing = false)
    @Configuration
    @Setter
    @Getter
    public static class InitConfig extends ProviderRouterInitConfig {

        // vertex.url.stream:https://#location-aiplatform.googleapis.com/v1/projects/#projectid/locations/#location/publishers/google/models/#model:streamGenerateContent
        @Value("${vertex.url.stream:https://aiplatform.googleapis.com/v1/projects/#projectid/locations/global/publishers/google/models/#model:streamGenerateContent}")
        // Vertex Stream API URL
        protected String urlStream;

        // vertex.url.once:https://#location-aiplatform.googleapis.com/v1/projects/#projectid/locations/#location/publishers/google/models/#model:generateContent
        @Value("${vertex.url.once:https://aiplatform.googleapis.com/v1/projects/#projectid/locations/global/publishers/google/models/#model:generateContent}")
        // Vertex非流式API URL
        protected String urlOnce;

        @Value("${vertex.project.id:}")
        // Vertex Project ID
        protected String projectId;

        @Value("${vertex.location:us-central1}")
        // Vertex API Location
        protected String location;

        @Bean(name = VertexRouter.NAME)
        @ConditionalOnMissingBean(name = VertexRouter.NAME)
        public VertexRouter vertexRouter() throws Exception {
            VertexRouter vertexRouter = new VertexRouter();
            BeanUtils.copyProperties(this, vertexRouter);
            log.info("VertexRouter inited. urlStream={},urlOnce={},location={},projectId={}", vertexRouter.getUrlStream(), vertexRouter.getUrlOnce(), vertexRouter.getLocation(), vertexRouter.getProjectId());
            return vertexRouter;
        }
    }
}
