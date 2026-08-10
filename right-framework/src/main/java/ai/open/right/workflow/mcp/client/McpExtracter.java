package ai.open.right.workflow.mcp.client;

import ai.open.right.utils.JsonUtils;
import org.apache.commons.collections.MapUtils;
import org.springframework.util.Assert;

import java.util.List;
import java.util.Map;

public class McpExtracter {

    public static <T> T getFirstText(McpResult<List<Map<String, Object>>> mcpResult, Class<T> clazz) throws Exception {
        return McpExtracter.getFirstText(mcpResult.getResult(), clazz);
    }

    public static <T> T getFirstText(List<Map<String, Object>> mcpResult, Class<T> clazz) throws Exception {
        String json = McpExtracter.getFirstText(mcpResult);
        return JsonUtils.read(JsonUtils.extract(json), clazz);
    }

    public static String getFirstText(McpResult<List<Map<String, Object>>> mcpResult) throws Exception {
        return McpExtracter.getFirstText(mcpResult.getResult(), String.class);
    }

    public static String getFirstText(List<Map<String, Object>> mcpResult) throws Exception {
        Assert.notEmpty(mcpResult, "MCP result object can not be empty");
        String text = MapUtils.getString(mcpResult.getFirst(), "text");
        Assert.hasText(text, "MCP result text can not be empty");
        return text;
    }

    public static McpData getFirstData(McpResult<List<Map<String, Object>>> mcpResult) throws Exception {
        return McpExtracter.getFirstData(mcpResult.getResult());
    }

    public static McpData getFirstData(List<Map<String, Object>> mcpResult) throws Exception {
        String json = McpExtracter.getFirstText(mcpResult);
        return new McpData(JsonUtils.read(JsonUtils.extract(json), Map.class));
    }

    public static <T> T getText(McpResult<List<Map<String, Object>>> mcpResult, Integer index, Class<T> clazz) throws Exception {
        return McpExtracter.getText(mcpResult.getResult(), index, clazz);
    }

    public static <T> T getText(List<Map<String, Object>> mcpResult, Integer index, Class<T> clazz) throws Exception {
        String json = McpExtracter.getText(mcpResult, index);
        return JsonUtils.read(JsonUtils.extract(json), clazz);
    }

    public static String getText(McpResult<List<Map<String, Object>>> mcpResult, Integer index) throws Exception {
        return McpExtracter.getText(mcpResult.getResult(), index);
    }

    public static String getText(List<Map<String, Object>> mcpResult, Integer index) throws Exception {
        Assert.notEmpty(mcpResult, "MCP result object can not be empty");
        String text = MapUtils.getString(mcpResult.get(index), "text");
        Assert.hasText(text, "MCP result text can not be empty");
        return text;
    }

    public static McpData getData(McpResult<List<Map<String, Object>>> mcpResult, Integer index) throws Exception {
        return McpExtracter.getData(mcpResult.getResult(), index);
    }

    public static McpData getData(List<Map<String, Object>> mcpResult, Integer index) throws Exception {
        String json = McpExtracter.getText(mcpResult, index);
        return new McpData(JsonUtils.read(JsonUtils.extract(json), Map.class));
    }
}
