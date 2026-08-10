package ai.open.right.workflow.flow.llm.provider.coze;

import org.junit.Assert;
import org.junit.Test;

public class CozeRequestTest {

    @Test
    public void test() {
        CozeRequest req = new CozeRequest();
        req.setBotId("BotID");
        Assert.assertEquals("BotID", req.getBotId());
    }

    @Test
    public void testGetResponseSchemaInheritsDefaultNull() {
        Assert.assertNull(new CozeRequest().getResponseSchema());
    }

    @Test
    public void testHashCode() throws Exception {
        Object object = CozeRequest.class.getConstructor(null).newInstance(null);
        Assert.assertEquals(object.hashCode(), object.hashCode());
        Assert.assertEquals(object, object);
    }
}
