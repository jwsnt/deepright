package ai.open.right.workflow.mcp.client.stream;

import ai.open.right.workflow.mcp.client.dimension.McpDimension;
import ai.open.right.workflow.mcp.client.McpIOReader;
import ai.open.right.workflow.mcp.client.McpIOWriter;
import lombok.extern.slf4j.Slf4j;
import org.apache.commons.io.IOUtils;
import org.apache.http.Header;
import org.apache.http.HttpResponse;
import org.apache.http.client.methods.HttpDelete;
import org.apache.http.client.methods.HttpPost;
import org.apache.http.client.methods.HttpRequestBase;
import org.apache.http.entity.StringEntity;
import org.apache.http.impl.nio.client.CloseableHttpAsyncClient;
import org.springframework.util.Assert;
import org.springframework.util.CollectionUtils;
import org.springframework.util.StringUtils;

import java.io.BufferedInputStream;
import java.io.IOException;
import java.io.InputStream;
import java.nio.charset.StandardCharsets;
import java.util.Arrays;
import java.util.Map;

@Slf4j
public class McpStreamHandler implements McpIOReader, McpIOWriter {

    public static final String ACCEPT = "application/json,text/event-stream";

    public static final String CONTENT_TYPE = "application/json";

    protected final CloseableHttpAsyncClient client;

    protected final Map<String, String> headers;

    protected final String http;

    protected String response;

    protected String request;

    protected String session;

    public McpStreamHandler(CloseableHttpAsyncClient client, Map<String, String> headers, String http) {
        this.headers = headers;
        this.client = client;
        this.http = http;
    }

    protected void response(HttpRequestBase request) throws Exception {
        HttpResponse response = this.execute(request);
        int statusCode = response.getStatusLine().getStatusCode();
        Assert.isTrue(!(statusCode >= 500 && statusCode <= 599), "MCP http server is error: " + statusCode);
        try (InputStream input = new BufferedInputStream(response.getEntity().getContent())) {
            this.session(response);
            this.response = IOUtils.toString(input, StandardCharsets.UTF_8);
            if (log.isDebugEnabled()) {
                log.debug("Mcp http response={}", this.response);
            }
            this.response = this.response.substring(Math.max(this.response.indexOf('{'), 0));
        }
    }

    protected HttpResponse execute(HttpRequestBase request) throws Exception {
        return this.client.execute(request, null).get();
    }

    protected void session(HttpResponse response) {
        Header session = response.getFirstHeader("Mcp-Session-Id");
        if (session != null) {
            this.session = session.getValue();
            if (log.isDebugEnabled()) {
                log.debug("Mcp http session={}", this.session);
            }
        }
    }

    @Override
    public void write(String content) throws Exception {
        this.request = StringUtils.hasText(this.request) ? this.request + content : content;
        if (log.isDebugEnabled()) {
            log.debug("Mcp http request={}", this.request);
        }
    }

    @Override
    public String readLine() throws Exception {
        try {
            Assert.hasText(this.response, "Mcp response can not be empty");
            return this.response;
        } finally {
            this.response = null;
        }
    }

    @Override
    public void flush(McpDimension dimension) throws Exception {
        try {
            HttpPost post = new HttpPost(this.http);
            post.setEntity(new StringEntity(this.request, StandardCharsets.UTF_8));
            post.setHeader("Content-Type", McpStreamHandler.CONTENT_TYPE);
            post.setHeader("Accept", McpStreamHandler.ACCEPT);
            this.additionalHeaders(dimension, post);
            if (StringUtils.hasText(this.session)) {
                post.setHeader("Mcp-Session-Id", this.session);
            }
            if (!CollectionUtils.isEmpty(this.headers)) {
                for (String key : this.headers.keySet()) {
                    post.setHeader(key, this.headers.get(key));
                }
            }
            if (log.isDebugEnabled()) {
                log.debug("Mcp http request: uri{},method={},headers={},request={}", post.getURI(), post.getMethod(), Arrays.toString(post.getAllHeaders()), this.request);
            }
            this.response(post);
        } finally {
            this.request = null;
        }
    }

    @Override
    public void close() throws IOException {
        if (StringUtils.hasText(this.session)) {
            HttpDelete delete = new HttpDelete(this.http);
            delete.setHeader("Content-Type", McpStreamHandler.CONTENT_TYPE);
            delete.setHeader("Accept", McpStreamHandler.ACCEPT);
            delete.setHeader("Mcp-Session-Id", this.session);
            try {
                if (log.isDebugEnabled()) {
                    log.debug("Mcp http request: uri={},method={},headers={}", delete.getURI(), delete.getMethod(), Arrays.toString(delete.getAllHeaders()));
                }
                this.response(delete);
            } catch (Exception e) {
                throw new IOException(e);
            }
        }
    }

    // 附加Header
    protected void additionalHeaders(McpDimension dimension, HttpPost post) {
        if (dimension != null) {
            post.setHeader("right_workflow", dimension.getWorkflow());
            post.setHeader("right_device", dimension.getDevice());
            post.setHeader("right_chat", dimension.getChat());
            post.setHeader("right_biz", dimension.getBiz());
            // 自定义Header
            if (!CollectionUtils.isEmpty(dimension.getHeaders())) {
                for (String key : dimension.getHeaders().keySet()) {
                    post.setHeader(key, dimension.getHeaders().get(key));
                }
            }
        }
    }
}
