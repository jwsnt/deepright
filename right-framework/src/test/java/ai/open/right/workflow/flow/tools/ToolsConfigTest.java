package ai.open.right.workflow.flow.tools;

import ai.open.right.protocol.ProtocolCode;
import ai.open.right.workflow.flow.config.TimeoutConfig;
import org.junit.Assert;
import org.junit.Test;

import java.util.List;

public class ToolsConfigTest {

    @Test
    public void test() {
        TimeoutConfig timeoutConfig = new TimeoutConfig();
        timeoutConfig.setTimeout4Service(1000);
        timeoutConfig.setTimeout4Llm(2000);
        ToolsConfig toolsConfig = new ToolsConfig();
        toolsConfig.setSource(false);
        toolsConfig.setTimeout(timeoutConfig);
        Assert.assertEquals(Integer.valueOf(1000), toolsConfig.getTimeout4Service(200));
        Assert.assertEquals(Integer.valueOf(2000), toolsConfig.getTimeout4Llm(200));
        Assert.assertEquals(timeoutConfig, toolsConfig.getTimeout());
        Assert.assertFalse(toolsConfig.getSource());
    }


    @Test
    public void testSuccessCode() {
        ToolsConfig toolsConfig = new ToolsConfig();
        toolsConfig.setSuccessCode(500);
        Assert.assertEquals(ProtocolCode.C500, toolsConfig.getSuccessCode());
    }

    @Test
    public void testMerge() throws Exception {
        ToolsConfig original = new ToolsConfig();
        original.setToolsOrchestrator(new ToolsOrchestrator());
        original.setSuccessCode(200);
        original.setService("originalService");
        original.setMethod("GET");
        original.setWrap(ToolsConfig.WRAP_OBJECT);
        original.setExpired(3600);
        original.setSource(true);
        TimeoutConfig originalTimeout = new TimeoutConfig();
        originalTimeout.setTimeout4Service(1000);
        original.setTimeout(originalTimeout);
        ToolsHeader originalHeader = new ToolsHeader();
        original.setHeaders(List.of(originalHeader));
        ToolsConfig other = new ToolsConfig();
        other.setToolsOrchestrator(new ToolsOrchestrator());
        other.setSuccessCode(500);
        other.setService("otherService");
        other.setMethod("POST");
        other.setWrap(ToolsConfig.WRAP_STRING);
        other.setExpired(7200);
        other.setSource(false);
        TimeoutConfig otherTimeout = new TimeoutConfig();
        otherTimeout.setTimeout4Llm(2000);
        other.setTimeout(otherTimeout);
        ToolsHeader otherHeader = new ToolsHeader();
        other.setHeaders(List.of(otherHeader));
        ToolsConfig merged = original.merge(other);
        Assert.assertEquals(original.getToolsOrchestrator(), merged.getToolsOrchestrator());
        Assert.assertEquals(original.getSuccessCode(), merged.getSuccessCode());
        Assert.assertEquals(original.getService(), merged.getService());
        Assert.assertEquals(original.getMethod(), merged.getMethod());
        Assert.assertEquals(original.getWrap(), merged.getWrap());
        Assert.assertEquals(original.getExpired(), merged.getExpired());
        Assert.assertEquals(original.getSource(), merged.getSource());
        Assert.assertEquals(original.getHeaders(), merged.getHeaders());
        Assert.assertEquals(originalTimeout.getTimeout4Service(), merged.getTimeout().getTimeout4Service(null));
        Assert.assertEquals(otherTimeout.getTimeout4Llm(), merged.getTimeout().getTimeout4Llm(null));
        ToolsConfig emptyOriginal = new ToolsConfig();
        ToolsConfig merged2 = emptyOriginal.merge(other);
        Assert.assertEquals(other.getToolsOrchestrator(), merged2.getToolsOrchestrator());
        Assert.assertEquals(other.getSuccessCode(), merged2.getSuccessCode());
        Assert.assertEquals(other.getService(), merged2.getService());
        Assert.assertEquals(other.getMethod(), merged2.getMethod());
        Assert.assertEquals(other.getWrap(), merged2.getWrap());
        Assert.assertEquals(other.getExpired(), merged2.getExpired());
        Assert.assertEquals(other.getSource(), merged2.getSource());
        Assert.assertEquals(other.getHeaders(), merged2.getHeaders());
        Assert.assertEquals(otherTimeout.getTimeout4Service(null), merged2.getTimeout().getTimeout4Service(null));
        Assert.assertEquals(otherTimeout.getTimeout4Llm(null), merged2.getTimeout().getTimeout4Llm(null));
        ToolsConfig result = original.merge(null);
        Assert.assertEquals(original, result);
    }
    @Test
    public void testGetMethodDefault() {
        ToolsConfig config = new ToolsConfig();
        config.setMethod(null);
        Assert.assertEquals("POST", config.getMethod());
    }

    @Test
    public void testIsValidWrap() {
        ToolsConfig config = new ToolsConfig();
        config.setWrap("object");
        Assert.assertTrue(config.isValidWrap());
        config.setWrap("invalid");
        Assert.assertFalse(config.isValidWrap());
    }
}
