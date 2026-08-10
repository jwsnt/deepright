package ai.open.right.workflow.flow.llm.provider.volcengine;

import ai.open.right.ObjectBuilder;
import ai.open.right.workflow.flow.llm.MessageDelegate;
import ai.open.right.workflow.flow.llm.config.LLMConfig;
import ai.open.right.workflow.flow.llm.provider.openai.OpenAiRequest;
import org.junit.Assert;
import org.junit.Test;

public class VolcengineRouterTest {

    @Test
    public void testHashCode1() throws Exception {
        Object object = VolcengineRouter.class.getConstructor(null).newInstance(null);
        Assert.assertEquals(object.hashCode(), object.hashCode());
        Assert.assertEquals(object, object);
    }

    @Test
    public void testHashCode2() throws Exception {
        Object object = VolcengineRouter.InitConfig.class.getConstructor(null).newInstance(null);
        Assert.assertEquals(object.hashCode(), object.hashCode());
        Assert.assertEquals(object, object);
    }

    @Test
    public void test() throws Exception {
        VolcengineRouter router = new VolcengineRouter();
        OpenAiRequest request = new OpenAiRequest();
        request.setMessage(new MessageDelegate(ObjectBuilder.buildLLMQuery()));
        router.setUrl("Hello World");
        Assert.assertEquals("Hello World", router.url(request, new LLMConfig(), ""));
    }
}
