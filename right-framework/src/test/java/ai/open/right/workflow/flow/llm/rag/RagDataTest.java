package ai.open.right.workflow.flow.llm.rag;

import ai.open.right.ObjectBuilder;
import ai.open.right.workflow.flow.llm.LLMQuery;
import org.junit.Assert;
import org.junit.Test;

public class RagDataTest {

    @Test
    public void test() {
        LLMQuery llmQuery = ObjectBuilder.buildLLMQuery();
        RagData data = RagData.builder()
                .query(llmQuery)
                .build();
        Assert.assertEquals(data.getQuery(), llmQuery);
    }
}
