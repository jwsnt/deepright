package ai.open.right.workflow.flow.script.impl;

import org.junit.Assert;
import org.junit.Test;

public class PolyglotContextTest {

    @Test
    public void test() throws Exception {
        PolyglotService.PolyglotContext polyglotContext = new PolyglotService.PolyglotContext(false);
        polyglotContext.eval("print(1+1)");
        Assert.assertEquals("2\n", polyglotContext.content());
        Assert.assertNotNull(polyglotContext.getContext());
        Assert.assertNotNull(polyglotContext.getStream());
        polyglotContext.setContext(null);
        polyglotContext.setStream(null);
        Assert.assertNull(polyglotContext.getContext());
        Assert.assertNull(polyglotContext.getStream());
    }
}
