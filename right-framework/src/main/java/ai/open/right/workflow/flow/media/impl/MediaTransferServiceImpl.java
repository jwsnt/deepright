package ai.open.right.workflow.flow.media.impl;

import ai.open.right.WorkflowException;
import ai.open.right.resouce.ResourceService;
import ai.open.right.workflow.flow.WorkflowTask;
import ai.open.right.workflow.flow.media.MediaConfig;
import ai.open.right.workflow.flow.media.MediaContext;
import ai.open.right.workflow.flow.media.MediaTransferService;
import ai.open.right.workflow.flow.media.MediaTransferUtils;
import jakarta.annotation.PostConstruct;
import lombok.Builder;
import lombok.Getter;
import lombok.Setter;
import lombok.extern.slf4j.Slf4j;
import org.apache.http.HttpResponse;
import org.apache.http.HttpStatus;
import org.apache.http.client.methods.HttpGet;
import org.apache.http.client.methods.HttpRequestBase;
import org.apache.http.impl.nio.client.CloseableHttpAsyncClient;
import org.springframework.beans.BeanUtils;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.boot.autoconfigure.condition.ConditionalOnMissingBean;
import org.springframework.boot.autoconfigure.condition.ConditionalOnProperty;
import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Configuration;
import org.springframework.util.Assert;

import java.io.BufferedInputStream;
import java.io.InputStream;
import java.util.ArrayList;
import java.util.Base64;
import java.util.List;
import java.util.concurrent.Future;

@Setter
@Getter
@Slf4j
public class MediaTransferServiceImpl implements MediaTransferService {

    protected CloseableHttpAsyncClient resource;

    protected ResourceService resourceService;

    protected MediaConfig defConfig;

    @PostConstruct
    public void init() throws Exception {
        this.defConfig = new MediaConfig();
        this.defConfig.setBase64(true);
    }

    public void transfer(MediaConfig mediaConfig, WorkflowTask workTask, List<MediaContext> mediaContext) throws Exception {
        if (mediaConfig.getBase64()) {
            List<MediaResource> resources = this.buildMediaContext(workTask, mediaContext);
            this.initMediaContext(mediaConfig, workTask, mediaContext, resources);
            if (log.isInfoEnabled()) {
                log.info("Media transfer size={}", resources.size());
            }
        }
    }

    @Override
    public void transfer(WorkflowTask workTask, List<MediaContext> mediaContext) throws Exception {
        this.transfer(this.defConfig, workTask, mediaContext);
    }

    protected void initMediaContext(MediaConfig mediaConfig, WorkflowTask workTask, List<MediaContext> mediaContext, List<MediaResource> resources) throws Exception {
        for (MediaResource resource : resources) {
            try {
                resource.init();
            } catch (Exception e) {
                WorkflowException.dolog(e);
            }
        }
    }

    protected List<MediaResource> buildMediaContext(WorkflowTask workTask, List<MediaContext> mediaContext) throws Exception {
        List<MediaResource> resources = new ArrayList<MediaResource>();
        for (MediaContext each : mediaContext) {
            if (each.canEncodeBase64()) {
                if (MediaTransferUtils.isNetwork(each.getData())) {
                    resources.add(this.buildHttpResource(each, workTask));
                } else {
                    resources.add(this.buildUriResource(each, workTask));
                }
            }
        }
        return resources;
    }

    protected Future<HttpResponse> getResponse(HttpRequestBase httpRequestBase, WorkflowTask workTask) throws Exception {
        return this.resource.execute(httpRequestBase, null);
    }

    protected MediaResource buildHttpResource(MediaContext mediaContext, WorkflowTask workTask) throws Exception {
        return MediaHttpResource.builder()
                .futureResponse(this.getResponse(new HttpGet(mediaContext.getData()), workTask))
                .mediaContext(mediaContext)
                .build();
    }

    protected MediaResource buildUriResource(MediaContext mediaContext, WorkflowTask workTask) throws Exception {
        return MediaURIResource.builder()
                .resourceService(this.resourceService)
                .uri(mediaContext.getData())
                .mediaContext(mediaContext)
                .build();
    }

    @Builder
    @Slf4j
    public static class MediaHttpResource implements MediaResource {

        protected final Future<HttpResponse> futureResponse;

        protected final MediaContext mediaContext;

        public void init() throws Exception {
            HttpResponse response = this.futureResponse.get();
            Assert.isTrue(response.getStatusLine().getStatusCode() == HttpStatus.SC_OK, "Media transfer request has failed: " + response.getStatusLine().getStatusCode());
            try (InputStream input = response.getEntity().getContent()) {
                this.mediaContext.setData(Base64.getEncoder().encodeToString((input.readAllBytes())));
                this.mediaContext.setType(MediaContext.PREFIX_INLINE + this.mediaContext.getType());
            }
        }
    }

    @Builder
    @Slf4j
    public static class MediaURIResource implements MediaResource {

        protected final ResourceService resourceService;

        protected final MediaContext mediaContext;

        protected final String uri;

        public void init() throws Exception {
            try (InputStream input = new BufferedInputStream(this.resourceService.url(this.uri).openStream())) {
                this.mediaContext.setData(Base64.getEncoder().encodeToString((input.readAllBytes())));
                this.mediaContext.setType(MediaContext.PREFIX_INLINE + this.mediaContext.getType());
            }
        }
    }

    public interface MediaResource {

        public void init() throws Exception;
    }

    @ConditionalOnProperty(name = "media.enable", havingValue = "true", matchIfMissing = false)
    @Configuration
    @Setter
    @Getter
    public static class InitConfig {

        @Autowired
        protected CloseableHttpAsyncClient resource;

        @Autowired
        protected ResourceService resourceService;

        @Bean
        @ConditionalOnMissingBean(value = MediaTransferService.class)
        public MediaTransferService mediaTransferService() throws Exception {
            MediaTransferServiceImpl mediaTransferService = new MediaTransferServiceImpl();
            BeanUtils.copyProperties(this, mediaTransferService);
            log.info("MediaTransferServiceImpl inited");
            return mediaTransferService;
        }
    }
}
