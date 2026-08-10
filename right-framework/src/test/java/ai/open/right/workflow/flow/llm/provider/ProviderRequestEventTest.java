package ai.open.right.workflow.flow.llm.provider;

import ai.open.right.ObjectBuilder;
import ai.open.right.workflow.flow.llm.Message;
import ai.open.right.workflow.flow.llm.MessageDelegate;
import org.junit.Assert;
import org.junit.Test;

public class ProviderRequestEventTest {

    @Test
    public void test() throws Exception {
        ProviderRequest providerRequest = new ProviderRequest();
        providerRequest.appendRequest("REQUEST");
        ProviderEvent providerRequestEvent = new ProviderEvent(providerRequest);
        Assert.assertEquals(ProviderEvent.TYPE, providerRequestEvent.getType());
        Assert.assertEquals("REQUEST", ProviderData.class.cast(providerRequestEvent.init().getData()).getRequest());
    }

    @Test
    public void testGet() {
        ProviderRequest providerRequest = new ProviderRequest();
        Message message = new MessageDelegate(ObjectBuilder.buildLLMQuery());
        providerRequest.setMessage(message);
        ProviderEvent providerRequestEvent = new ProviderEvent(providerRequest);
        Assert.assertEquals(message.getDevice(), providerRequestEvent.getDevice());
        Assert.assertEquals(ProviderEvent.TYPE, providerRequestEvent.getType());
        Assert.assertEquals(message.getBiz(), providerRequestEvent.getBiz());
        Assert.assertEquals(message.getChat(), providerRequestEvent.getChat());
        Assert.assertEquals(message.getCreated(), providerRequestEvent.getNow());

    }
}
