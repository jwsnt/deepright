package ai.open.right.workflow.flow.llm.provider;

import ai.open.right.ObjectBuilder;
import ai.open.right.workflow.flow.llm.MessageDelegate;
import org.junit.Assert;
import org.junit.Test;

public class ProviderEventTest {

    @Test
    public void test() {
        ProviderRequest r = new ProviderRequest();
        r.setMessage(new MessageDelegate(ObjectBuilder.buildLLMQuery()));
        ProviderEvent providerEvent = new ProviderEvent(r);
        Assert.assertEquals("UNKNOWN-UNKNOWN-UNKNOWN", providerEvent.getDimension());
        Assert.assertEquals(r.getMessage().getWorkflow(), providerEvent.getWorkflow());
        Assert.assertNotNull(providerEvent.getProviderRequest());
        Assert.assertNotNull(providerEvent.getProviderData());
    }
}
