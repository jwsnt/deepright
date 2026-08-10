package ai.open.right.workflow.mcp.client;

import ai.open.right.WorkflowException;
import ai.open.right.protocol.ProtocolCode;
import ai.open.right.utils.JsonUtils;
import ai.open.right.workflow.mcp.client.dimension.McpDimension;
import com.google.common.collect.ImmutableMap;
import lombok.Getter;
import lombok.Setter;
import lombok.extern.slf4j.Slf4j;
import org.apache.commons.io.IOUtils;
import org.apache.commons.lang3.StringUtils;
import org.apache.commons.lang3.builder.ToStringBuilder;
import org.springframework.util.Assert;
import org.springframework.util.CollectionUtils;

import java.io.Closeable;
import java.io.IOException;
import java.util.Collections;
import java.util.HashMap;
import java.util.List;
import java.util.Map;

@Slf4j
@Getter
@Setter
abstract public class McpClient implements Closeable {

    protected static final List<Map<String, Object>> EMPTY_TOOLS = Collections.emptyList();

    protected static final Map<String, Object> EMPTY_PROMPT = Collections.emptyMap();

    protected static final Map<String, Object> INITIAL = new HashMap<>();

    public static final String VERSION = "2025-06-18";

    static {
        McpClient.INITIAL.put("capabilities", ImmutableMap.<String, String>builder());
        McpClient.INITIAL.put("clientInfo", ImmutableMap.<String, String>builder().put("version", "1.0.0").put("name", "right").build());
        McpClient.INITIAL.put("protocolVersion", McpClient.VERSION);
    }

    protected McpIOWriter stdOutput;

    protected McpIOReader stdInput;

    public List<Map<String, Object>> toolsCall(String name, Map<String, Object> arguments, McpDimension dimension) throws Exception {
        if (log.isDebugEnabled()) {
            log.debug("Start tools call={}-{}", name, arguments);
        }
        McpResponse response = this.request(dimension, new McpRequest(McpProtocol.PROTOCOL_TOOLS_CALL, ImmutableMap.<String, Object>builder().put("arguments", arguments).put("name", name).build()));
        if (log.isInfoEnabled()) {
            log.info("Finish tools call={}", ToStringBuilder.reflectionToString(response));
        }
        Map<String, Object> result = Map.class.cast(response.getResult());
        return CollectionUtils.isEmpty(result) ? McpClient.EMPTY_TOOLS : List.class.cast(result.get("content"));
    }

    public List<Map<String, Object>> toolsList(McpDimension dimension) throws Exception {
        if (log.isDebugEnabled()) {
            log.debug("Start tools list");
        }
        McpResponse response = this.request(dimension, new McpRequest(McpProtocol.PROTOCOL_TOOLS_LIST));
        Map<String, Object> result = Map.class.cast(response.getResult());
        return !CollectionUtils.isEmpty(result) ? List.class.cast(result.get("tools")) : null;
    }

    public List<Map<String, Object>> promptList(McpDimension dimension) throws Exception {
        if (log.isDebugEnabled()) {
            log.debug("Start prompt list");
        }
        McpResponse response = this.request(dimension, new McpRequest(McpProtocol.PROTOCOL_PROMPTS_LIST));
        if (log.isInfoEnabled()) {
            log.info("Finish prompt list={}", response);
        }
        Map<String, Object> result = Map.class.cast(response.getResult());
        return !CollectionUtils.isEmpty(result) ? List.class.cast(result.get("prompts")) : null;
    }

    public Map<String, Object> promptGet(String name, Map<String, Object> arguments, McpDimension dimension) throws Exception {
        if (log.isDebugEnabled()) {
            log.debug("Start prompt get={}-{}", name, arguments);
        }
        McpResponse response = this.request(dimension, new McpRequest(McpProtocol.PROTOCOL_PROMPTS_GET, ImmutableMap.<String, Object>builder().put("arguments", CollectionUtils.isEmpty(arguments) ? McpClient.EMPTY_PROMPT : arguments).put("name", name).build()));
        if (log.isInfoEnabled()) {
            log.info("Finish prompt get={}", response);
        }
        return Map.class.cast(response.getResult());
    }

    public List<Map<String, Object>> resourcesTemplatesList(McpDimension dimension) throws Exception {
        if (log.isDebugEnabled()) {
            log.debug("Start resources templates list");
        }
        McpResponse response = this.request(dimension, new McpRequest(McpProtocol.PROTOCOL_RESOURCES_TEMPLATES_LIST));
        if (log.isDebugEnabled()) {
            log.debug("Finish resources templates list={}", response);
        }
        Map<String, Object> result = Map.class.cast(response.getResult());
        return !CollectionUtils.isEmpty(result) ? List.class.cast(result.get("resourceTemplates")) : null;
    }

    public List<Map<String, Object>> resourcesList(McpDimension dimension) throws Exception {
        if (log.isDebugEnabled()) {
            log.debug("Start resources list");
        }
        McpResponse response = this.request(dimension, new McpRequest(McpProtocol.PROTOCOL_RESOURCES_LIST));
        if (log.isDebugEnabled()) {
            log.debug("Finish resources list={}", response);
        }
        Map<String, Object> result = Map.class.cast(response.getResult());
        return !CollectionUtils.isEmpty(result) ? List.class.cast(result.get("resources")) : null;
    }

    public List<Map<String, Object>> resourcesRead(String uri, McpDimension dimension) throws Exception {
        if (log.isDebugEnabled()) {
            log.debug("Start resources read={}", uri);
        }
        McpResponse response = this.request(dimension, new McpRequest(McpProtocol.PROTOCOL_RESOURCES_READ, Collections.singletonMap("uri", uri)));
        if (log.isDebugEnabled()) {
            log.debug("Finish resources read={}", response);
        }
        Map<String, Object> result = Map.class.cast(response.getResult());
        return !CollectionUtils.isEmpty(result) ? List.class.cast(result.get("contents")) : null;
    }

    protected McpResponse request(McpDimension dimension, McpRequest request, Boolean interrupt) throws Exception {
        String jsonRpc = JsonUtils.write(request);
        if (log.isDebugEnabled()) {
            log.debug("Mcp request={}", jsonRpc);
        }
        this.stdOutput.write(jsonRpc);
        this.stdOutput.write(System.lineSeparator());
        // 刷新
        this.stdOutput.flush(dimension);
        if (request.fetchResponse()) {
            return this.response(request, interrupt);
        } else {
            return null;
        }
    }

    protected McpResponse response(McpRequest request, Boolean interrupt) throws Exception {
        StringBuffer response = null;
        for (; ; ) {
            if (Thread.currentThread().isInterrupted()) {
                throw new InterruptedException("MCP response reading interrupted");
            }
            String current = this.stdInput.readLine();
            if (log.isDebugEnabled()) {
                log.debug("Mcp server response={}", current);
            }
            // 如果解析为JSON则立即返回
            if (!JsonUtils.like(current)) {
                (response = response != null ? response : new StringBuffer()).append(!StringUtils.isEmpty(current) ? current : "");
            } else {
                return JsonUtils.read(current, McpResponse.class).check(interrupt);
            }
            if (current == null) {
                throw new WorkflowException("Mcp server response cannot be parsed, " + request.getProtocol() + ": " + response, ProtocolCode.C500);
            }
            Thread.yield();
        }
    }

    protected McpResponse request(McpDimension dimension, McpRequest request) throws Exception {
        return this.request(dimension, request, false);
    }

    protected McpResponse request(McpRequest request, Boolean interrupt) throws Exception {
        return this.request(null, request, interrupt);
    }

    protected McpResponse request(McpRequest request) throws Exception {
        return this.request(null, request, false);
    }

    protected void init(String name) throws Exception {
        McpResponse response = this.request(new McpRequest(McpProtocol.PROTOCOL_INITIALIZE, McpClient.INITIAL), true);
        Map<String, Object> result = Map.class.cast(response.getResult());
        if (log.isInfoEnabled()) {
            log.info("Mcp server init={}", result);
        }
        Assert.notNull(result, "Result can not be empty");
        String protocol = result.get("protocolVersion").toString();
        Assert.hasText(protocol, "Protocol version is error");
        if (log.isDebugEnabled()) {
            Map<String, Object> serverInfo = Map.class.cast(result.get("serverInfo"));
            log.debug("Mcp server is ready={}", serverInfo);
        }
        this.request(new McpRequest(McpProtocol.PROTOCOL_INITIALIZED));
    }

    @Override
    public void close() throws IOException {
        IOUtils.closeQuietly(this.stdOutput);
        IOUtils.closeQuietly(this.stdInput);
    }
}
