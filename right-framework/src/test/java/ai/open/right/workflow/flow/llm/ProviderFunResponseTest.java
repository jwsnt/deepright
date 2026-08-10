package ai.open.right.workflow.flow.llm;

import ai.open.right.workflow.flow.llm.provider.ProviderFunCallResponse;
import org.junit.Assert;
import org.junit.Test;

public class ProviderFunResponseTest {

    @Test
    public void test() {
        ProviderFunCallResponse providerFunResponse = ProviderFunCallResponse.builder()
                .name("NAME")
                .response("RESP")
                .build();
        Assert.assertEquals("NAME",providerFunResponse.getName());
        Assert.assertEquals("RESP",providerFunResponse.getResponse());
    }
}
