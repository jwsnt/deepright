package ai.open.right.workflow.flow.llm.provider;

import org.junit.Assert;
import org.junit.Test;

public class ProviderFunDataTest {

    @Test
    public void testSetGet() {
        ProviderFunCallRequest providerFunRequest = ProviderFunCallRequest.builder().name("NAME").build();
        ProviderFunCallResponse providerFunResponse = ProviderFunCallResponse.builder().name("NAME").response("RESPONSE").build();
        ProviderFunCallData providerFunData = new ProviderFunCallData();
        providerFunData.addFunCall(providerFunRequest, providerFunResponse);
        Assert.assertEquals(providerFunRequest, providerFunData.getRequests().getFirst());
        Assert.assertEquals(providerFunResponse, providerFunData.getResponses().getFirst());

    }
}
