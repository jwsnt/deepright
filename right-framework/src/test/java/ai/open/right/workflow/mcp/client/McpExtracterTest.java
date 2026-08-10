package ai.open.right.workflow.mcp.client;

import lombok.Data;
import org.junit.Assert;
import org.junit.Test;

import java.util.ArrayList;
import java.util.HashMap;
import java.util.List;
import java.util.Map;

public class McpExtracterTest {

    @Test
    public void test0() throws Exception {
        Map<String, Object> data1 = new HashMap<String, Object>();
        data1.put("text", "{\"HELLO_1\":\"WORLD_1\"}");
        Map<String, Object> data2 = new HashMap<String, Object>();
        data1.put("text", "{\"HELLO_2\":\"WORLD_2\"}");
        List<Map<String, Object>> data = new ArrayList<Map<String, Object>>();
        data.add(data1);
        data.add(data2);
        McpResult<List<Map<String, Object>>> mcpResult = new McpResult<List<Map<String, Object>>>();
        mcpResult.setResult(data);
        Assert.assertEquals("{\"HELLO_2\":\"WORLD_2\"}", McpExtracter.getFirstText(mcpResult));
    }

    @Test
    public void test1() throws Exception {
        Map<String, Object> data1 = new HashMap<String, Object>();
        data1.put("text", "{\"HELLO_1\":\"WORLD_1\"}");
        Map<String, Object> data2 = new HashMap<String, Object>();
        data1.put("text", "{\"HELLO_2\":\"WORLD_2\"}");
        List<Map<String, Object>> data = new ArrayList<Map<String, Object>>();
        data.add(data1);
        data.add(data2);
        McpResult<List> mcpResult = new McpResult<List>();
        mcpResult.setResult(data);
        Assert.assertEquals("{\"HELLO_2\":\"WORLD_2\"}", McpExtracter.getFirstText(mcpResult.getResult()));
    }

    @Test
    public void test2() throws Exception {
        Map<String, Object> data1 = new HashMap<String, Object>();
        data1.put("text", "{\"HELLO_1\":\"WORLD_1\"}");
        Map<String, Object> data2 = new HashMap<String, Object>();
        data1.put("text", "{\"HELLO_2\":\"WORLD_2\"}");
        List<Map<String, Object>> data = new ArrayList<Map<String, Object>>();
        data.add(data1);
        data.add(data2);
        McpResult<List> mcpResult = new McpResult<List>();
        mcpResult.setResult(data);
        Assert.assertEquals("WORLD_2", McpExtracter.getFirstText(mcpResult.getResult(), DataPart.class).getHELLO_2());
    }

    @Test
    public void test3() throws Exception {
        Map<String, Object> data1 = new HashMap<String, Object>();
        data1.put("text", "{\"HELLO_1\":\"WORLD_1\"}");
        Map<String, Object> data2 = new HashMap<String, Object>();
        data1.put("text", "{\"HELLO_2\":\"WORLD_2\"}");
        List<Map<String, Object>> data = new ArrayList<Map<String, Object>>();
        data.add(data1);
        data.add(data2);
        McpResult<List> mcpResult = new McpResult<List>();
        mcpResult.setResult(data);
        Assert.assertEquals("{\"HELLO_2\":\"WORLD_2\"}", McpExtracter.getFirstText(mcpResult.getResult()));
    }

    @Test
    public void test4() throws Exception {
        Map<String, Object> data1 = new HashMap<String, Object>();
        data1.put("text", "{\"HELLO_1\":\"WORLD_1\"}");
        Map<String, Object> data2 = new HashMap<String, Object>();
        data1.put("text", "{\"HELLO_2\":\"WORLD_2\"}");
        List<Map<String, Object>> data = new ArrayList<Map<String, Object>>();
        data.add(data1);
        data.add(data2);
        McpResult<List> mcpResult = new McpResult<List>();
        mcpResult.setResult(data);
        Assert.assertEquals("WORLD_2", McpExtracter.getFirstData(mcpResult.getResult()).getObject("HELLO_2"));
    }

    @Test
    public void test5() throws Exception {
        Map<String, Object> data1 = new HashMap<String, Object>();
        data1.put("text", "{\"HELLO_1\":\"WORLD_1\"}");
        Map<String, Object> data2 = new HashMap<String, Object>();
        data1.put("text", "{\"HELLO_2\":\"WORLD_2\"}");
        List<Map<String, Object>> data = new ArrayList<Map<String, Object>>();
        data.add(data1);
        data.add(data2);
        McpResult<List> mcpResult = new McpResult<List>();
        mcpResult.setResult(data);
        Assert.assertEquals("{\"HELLO_2\":\"WORLD_2\"}", McpExtracter.getText(mcpResult.getResult(), 0));
    }

    @Test
    public void test6() throws Exception {
        Map<String, Object> data1 = new HashMap<String, Object>();
        data1.put("text", "{\"HELLO_1\":\"WORLD_1\"}");
        Map<String, Object> data2 = new HashMap<String, Object>();
        data1.put("text", "{\"HELLO_2\":\"WORLD_2\"}");
        List<Map<String, Object>> data = new ArrayList<Map<String, Object>>();
        data.add(data1);
        data.add(data2);
        McpResult<List<Map<String, Object>>> mcpResult = new McpResult<List<Map<String, Object>>>();
        mcpResult.setResult(data);
        Assert.assertEquals("WORLD_2", McpExtracter.getText(mcpResult, 0, DataPart.class).getHELLO_2());
    }

    @Test
    public void test7() throws Exception {
        Map<String, Object> data1 = new HashMap<String, Object>();
        data1.put("text", "{\"HELLO_1\":\"WORLD_1\"}");
        Map<String, Object> data2 = new HashMap<String, Object>();
        data1.put("text", "{\"HELLO_2\":\"WORLD_2\"}");
        List<Map<String, Object>> data = new ArrayList<Map<String, Object>>();
        data.add(data1);
        data.add(data2);
        McpResult<List> mcpResult = new McpResult<List>();
        mcpResult.setResult(data);
        Assert.assertEquals("WORLD_2", McpExtracter.getText(mcpResult.getResult(), 0, DataPart.class).getHELLO_2());
    }

    @Test
    public void test8() throws Exception {
        Map<String, Object> data1 = new HashMap<String, Object>();
        data1.put("text", "{\"HELLO_1\":\"WORLD_1\"}");
        Map<String, Object> data2 = new HashMap<String, Object>();
        data1.put("text", "{\"HELLO_2\":\"WORLD_2\"}");
        List<Map<String, Object>> data = new ArrayList<Map<String, Object>>();
        data.add(data1);
        data.add(data2);
        McpResult<List> mcpResult = new McpResult<List>();
        mcpResult.setResult(data);
        Assert.assertEquals("WORLD_2", McpExtracter.getData(mcpResult.getResult(), 0).getObject("HELLO_2"));
    }

    @Test
    public void test9() throws Exception {
        Map<String, Object> data1 = new HashMap<String, Object>();
        data1.put("text", "{\"HELLO_1\":\"WORLD_1\"}");
        Map<String, Object> data2 = new HashMap<String, Object>();
        data1.put("text", "{\"HELLO_2\":\"WORLD_2\"}");
        List<Map<String, Object>> data = new ArrayList<Map<String, Object>>();
        data.add(data1);
        data.add(data2);
        McpResult<List<Map<String, Object>>> mcpResult = new McpResult<List<Map<String, Object>>>();
        mcpResult.setResult(data);
        Assert.assertEquals("WORLD_2", McpExtracter.getData(mcpResult, 0).getObject("HELLO_2"));
    }

    @Test
    public void test10() throws Exception {
        Map<String, Object> data1 = new HashMap<String, Object>();
        data1.put("text", "{\"HELLO_1\":\"WORLD_1\"}");
        Map<String, Object> data2 = new HashMap<String, Object>();
        data1.put("text", "{\"HELLO_2\":\"WORLD_2\"}");
        List<Map<String, Object>> data = new ArrayList<Map<String, Object>>();
        data.add(data1);
        data.add(data2);
        McpResult<List<Map<String, Object>>> mcpResult = new McpResult<List<Map<String, Object>>>();
        mcpResult.setResult(data);
        Assert.assertEquals("WORLD_2", McpExtracter.getFirstText(mcpResult, DataPart.class).getHELLO_2());
    }

    @Test
    public void test11() throws Exception {
        Map<String, Object> data1 = new HashMap<String, Object>();
        data1.put("text", "{\"HELLO_1\":\"WORLD_1\"}");
        Map<String, Object> data2 = new HashMap<String, Object>();
        data1.put("text", "{\"HELLO_2\":\"WORLD_2\"}");
        List<Map<String, Object>> data = new ArrayList<Map<String, Object>>();
        data.add(data1);
        data.add(data2);
        McpResult<List<Map<String, Object>>> mcpResult = new McpResult<List<Map<String, Object>>>();
        mcpResult.setResult(data);
        Assert.assertEquals("WORLD_2", McpExtracter.getFirstData(mcpResult).getObject("HELLO_2"));
    }

    @Test
    public void test12() throws Exception {
        Map<String, Object> data1 = new HashMap<String, Object>();
        data1.put("text", "{\"HELLO_1\":\"WORLD_1\"}");
        Map<String, Object> data2 = new HashMap<String, Object>();
        data1.put("text", "{\"HELLO_2\":\"WORLD_2\"}");
        List<Map<String, Object>> data = new ArrayList<Map<String, Object>>();
        data.add(data1);
        data.add(data2);
        McpResult<List<Map<String, Object>>> mcpResult = new McpResult<List<Map<String, Object>>>();
        mcpResult.setResult(data);
        Assert.assertEquals("WORLD_2", McpExtracter.getText(mcpResult, 0, DataPart.class).getHELLO_2());
    }

    @Test
    public void test13() throws Exception {
        Map<String, Object> data1 = new HashMap<String, Object>();
        data1.put("text", "{\"HELLO_1\":\"WORLD_1\"}");
        Map<String, Object> data2 = new HashMap<String, Object>();
        data1.put("text", "{\"HELLO_2\":\"WORLD_2\"}");
        List<Map<String, Object>> data = new ArrayList<Map<String, Object>>();
        data.add(data1);
        data.add(data2);
        McpResult<List<Map<String, Object>>> mcpResult = new McpResult<List<Map<String, Object>>>();
        mcpResult.setResult(data);
        Assert.assertEquals("{\"HELLO_2\":\"WORLD_2\"}", McpExtracter.getText(mcpResult, 0));
    }

    
    public static class DataPart {

        private String HELLO_2;
        public String getHELLO_2() { return HELLO_2; }
        public void setHELLO_2(String HELLO_2) { this.HELLO_2 = HELLO_2; }

    }
}
