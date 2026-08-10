package ai.open.right.workflow.flow.llm;

import ai.open.right.workflow.flow.llm.provider.ProviderData;
import org.junit.Assert;
import org.junit.Test;

public class ProviderDataTest {

    @Test
    public void test() throws Exception {
        ProviderData providerData = new ProviderData();
        providerData.appendResponse("{\"KEY_A\":\"VAL_A\"}");
        providerData.appendRequest("{\"KEY_A\":\"VAL_B\"}");
        Assert.assertEquals("{\"KEY_A\":\"VAL_B\"}", providerData.getRequest());
        providerData.init();
        Assert.assertEquals("{\"KEY_A\":\"VAL_A\"}", providerData.getResponse());
        providerData.setRequest("A");
        providerData.setResponse("B");
        Assert.assertEquals("A", providerData.getRequest());
        Assert.assertEquals("B", providerData.getResponse());
    }

    @Test
    public void testResponseBuffer_appendAccumulatesBeforeInit() throws Exception {
        ProviderData data = new ProviderData();
        Assert.assertNotNull(data.getResponseBuffer());
        Assert.assertEquals(0, data.getResponseBuffer().length());
        data.appendResponse("a");
        data.appendResponse("bc");
        Assert.assertEquals("abc", data.getResponseBuffer().toString());
        Assert.assertEquals("", data.getResponse());
    }

    @Test
    public void testInit_flushesResponseBufferToResponseAndClearsBuffer() throws Exception {
        ProviderData data = new ProviderData();
        data.appendResponse("part1");
        data.appendResponse("part2");
        ProviderData same = data.init();
        Assert.assertSame(data, same);
        Assert.assertEquals("part1part2", data.getResponse());
        Assert.assertNotNull(data.getResponseBuffer());
    }

    @Test
    public void testInit_emptyBufferYieldsEmptyResponse() throws Exception {
        ProviderData data = new ProviderData();
        data.init();
        Assert.assertEquals("", data.getResponse());
        Assert.assertNotNull(data.getResponseBuffer());
    }

    @Test
    public void testInit_secondCallWithDoubleInit() throws Exception {
        ProviderData data = new ProviderData();
        data.appendResponse("x");
        data.init();
        data.init();
    }
}
