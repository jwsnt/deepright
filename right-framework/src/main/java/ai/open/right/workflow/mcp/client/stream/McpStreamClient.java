package ai.open.right.workflow.mcp.client.stream;

import ai.open.right.WorkflowException;
import ai.open.right.workflow.mcp.McpMethod;
import ai.open.right.workflow.mcp.client.McpClient;
import ai.open.right.workflow.mcp.client.dimension.McpDimension;
import com.google.common.cache.Cache;
import com.google.common.cache.CacheBuilder;
import lombok.Getter;
import lombok.Setter;
import lombok.extern.slf4j.Slf4j;
import org.apache.http.impl.nio.client.CloseableHttpAsyncClient;

import java.io.IOException;
import java.util.List;
import java.util.Map;
import java.util.concurrent.Callable;

@Slf4j
@Setter
@Getter
public class McpStreamClient extends McpClient {

    protected final Cache<String, List<Map<String, Object>>> cache = CacheBuilder.newBuilder().maximumSize(Integer.MAX_VALUE).build();

    protected McpStreamHandler handler;

    public McpStreamClient(CloseableHttpAsyncClient client, String name, String http, Map<String, String> headers) throws Exception {
        try {
            this.handler = new McpStreamHandler(client, headers, http);
            super.stdOutput = handler;
            super.stdInput = handler;
            this.init(name);
        } catch (Exception e) {
            WorkflowException.dolog(e);
            this.close();
            throw e;
        }
    }

    public List<Map<String, Object>> resourcesTemplatesList(McpDimension dimension) throws Exception {
        return this.cache.get(McpMethod.KEY_RESOURCES_TEMPLATES_LIST, new McpStreamClient.ResourceTemplateListCallable());
    }

    public List<Map<String, Object>> resourcesList(McpDimension dimension) throws Exception {
        return this.cache.get(McpMethod.KEY_RESOURCES_LIST, new McpStreamClient.ResourceListCallable());
    }

    @Override
    public List<Map<String, Object>> promptList(McpDimension dimension) throws Exception {
        return this.cache.get(McpMethod.KEY_PROMPTS_LIST, new McpStreamClient.PromptListCallable());
    }

    @Override
    public List<Map<String, Object>> toolsList(McpDimension dimension) throws Exception {
        return this.cache.get(McpMethod.KEY_TOOLS_LIST, new McpStreamClient.ToolsListCallable());
    }

    private class ResourceTemplateListCallable implements Callable<List<Map<String, Object>>> {

        @Override
        public List<Map<String, Object>> call() throws Exception {
            // 缓存，不支持McpDimension
            return McpStreamClient.super.resourcesTemplatesList(null);
        }
    }

    private class ResourceListCallable implements Callable<List<Map<String, Object>>> {

        @Override
        public List<Map<String, Object>> call() throws Exception {
            // 缓存，不支持McpDimension
            return McpStreamClient.super.resourcesList(null);
        }
    }

    private class PromptListCallable implements Callable<List<Map<String, Object>>> {

        @Override
        public List<Map<String, Object>> call() throws Exception {
            // 缓存，不支持McpDimension
            return McpStreamClient.super.promptList(null);
        }
    }

    private class ToolsListCallable implements Callable<List<Map<String, Object>>> {

        @Override
        public List<Map<String, Object>> call() throws Exception {
            // 缓存，不支持McpDimension
            return McpStreamClient.super.toolsList(null);
        }
    }

    @Override
    public void close() throws IOException {
        super.close();
        if (this.handler != null) {
            this.handler.close();
        }
    }
}
