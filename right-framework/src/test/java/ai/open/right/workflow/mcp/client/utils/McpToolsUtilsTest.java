package ai.open.right.workflow.mcp.client.utils;

import ai.open.right.utils.JsonUtils;
import ai.open.right.workflow.mcp.client.utils.McpToolsUtils;
import org.apache.commons.io.IOUtils;
import org.junit.Assert;
import org.junit.Test;
import org.springframework.util.ResourceUtils;

import java.util.ArrayList;
import java.util.Collections;
import java.util.List;
import java.util.Map;

public class McpToolsUtilsTest {

    @Test
    public void test() throws Exception {
        Map<String, Object> json = JsonUtils.read(IOUtils.toString(ResourceUtils.getURL("classpath:VertexFunCall_properties.json").openStream()), Map.class);
        List<Map<String, Object>> before = List.class.cast(json.get("function_declarations"));
        List<Map<String, Object>> after = new ArrayList<>();
        for (Map<String, Object> each : before) {
            Map<String, Object> parameters = Map.class.cast(each.get("parameters"));
            Map<String, Object> filter = McpToolsUtils.filter((Map<String, Object>) parameters.get("properties"));
            parameters.put("properties", filter);
            after.add(each);
        }
        String expect = IOUtils.toString(ResourceUtils.getURL("classpath:VertexFunCall_properties_after.json").openStream());
        Assert.assertEquals(expect, JsonUtils.write(Collections.singletonMap("function_declarations", after)));
    }
}
