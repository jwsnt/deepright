package ai.open.right.workflow.flow.tools;

import org.junit.Assert;
import org.junit.Test;

public class ToolsDeliveryTest {

    @Test
    public void test() {
        ToolsPackage toolsDelivery = ToolsPackage.builder().build();
        toolsDelivery.setQuery("HELLO");
        toolsDelivery.setTools("WORLD");
        Assert.assertEquals("HELLO", toolsDelivery.getQuery());
        Assert.assertEquals("WORLD", toolsDelivery.getTools());
    }
}
