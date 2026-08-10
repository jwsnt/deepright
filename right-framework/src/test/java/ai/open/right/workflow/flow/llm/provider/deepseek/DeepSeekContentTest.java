package ai.open.right.workflow.flow.llm.provider.deepseek;

import ai.open.right.workflow.flow.llm.LLMFunCallRequest;
import ai.open.right.workflow.flow.llm.LLMFunCallResponse;
import ai.open.right.workflow.flow.llm.provider.ProviderFunCallRequest;
import ai.open.right.workflow.flow.llm.provider.ProviderFunCallResponse;
import com.google.common.collect.ImmutableMap;
import org.junit.Assert;
import org.junit.Test;

public class DeepSeekContentTest {

    @Test
    public void test1() throws Exception {
        LLMFunCallRequest llmFunCallRequest = ProviderFunCallRequest.builder()
                .reason("content")
                .args(ImmutableMap.of("reasoning_content", "content"))
                .name("NAME")
                .build();
        Object ct = DeepSeekRouterReflectTestUtil.newOpenAiContent(llmFunCallRequest);
        Assert.assertEquals("content", DeepSeekRouterReflectTestUtil.getReasoning(ct));
    }

    @Test
    public void test2() throws Exception {
        LLMFunCallResponse llmFunCallResponse = ProviderFunCallResponse.builder().id("assistant").build();
        DeepSeekRouterReflectTestUtil.newOpenAiContent(llmFunCallResponse);
    }
}
