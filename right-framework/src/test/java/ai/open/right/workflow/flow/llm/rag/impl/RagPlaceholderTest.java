package ai.open.right.workflow.flow.llm.rag.impl;

import ai.open.right.ObjectBuilder;
import ai.open.right.resouce.PlaceholderResolver;
import ai.open.right.workflow.flow.llm.LLMQuery;
import ai.open.right.workflow.flow.llm.config.LLMConfig;
import ai.open.right.workflow.flow.llm.rag.RagConfig;
import ai.open.right.workflow.flow.llm.rag.RagData;
import ai.open.right.workflow.flow.llm.rag.future.RagFuture;
import ai.open.right.workflow.notify.impl.NotifierServiceImpl;
import org.easymock.EasyMock;
import org.junit.Assert;
import org.junit.Test;

public class RagPlaceholderTest {

    @Test
    public void test() throws Exception {
        RagPlaceholder ragPlaceholder = new RagPlaceholder();
        ragPlaceholder.setPlaceholderResolver(new PlaceholderResolver() {

            @Override
            public String replace(String input) throws Exception {
                return input.toLowerCase();
            }
        });
        LLMQuery query = ObjectBuilder.buildLLMQuery();
        RagData ragData = RagData.builder()
                .config(new LLMConfig())
                .prompt("UNKNOWN ${key}")
                .query(query)
                .build();
        RagConfig ragConfig = new RagConfig();
        ragPlaceholder.rag(ragConfig, ragData);
        String expected = "unknown ${key}";
        Assert.assertEquals(expected, ragData.getPrompt());
    }

    @Test
    public void testNotAllowed() throws Exception {
        RagPlaceholder ragPlaceholder = new RagPlaceholder() {
            protected Boolean allowed(RagConfig ragConfig, RagData ragData) throws Exception {
                return false;
            }
        };
        ragPlaceholder.setPlaceholderResolver(new PlaceholderResolver() {

            @Override
            public String replace(String input) throws Exception {
                return input;
            }
        });
        LLMQuery query = ObjectBuilder.buildLLMQuery();
        RagData ragData = RagData.builder()
                .config(new LLMConfig())
                .prompt("UNKNOWN #key")
                .query(query)
                .build();
        RagConfig ragConfig = new RagConfig();
        Assert.assertEquals(RagFuture.NOTHING, ragPlaceholder.rag(ragConfig, ragData));
    }

    @Test
    public void testInit() throws Exception {
        NotifierServiceImpl notifierManager = ObjectBuilder.buildActualNotifierManagerWithNothing();
        PlaceholderResolver placeholderResolver = EasyMock.createMock(PlaceholderResolver.class);
        EasyMock.replay(placeholderResolver);
        RagPlaceholder.InitConfig service = new RagPlaceholder.InitConfig();
        service.setNotifierService(notifierManager);
        service.setPlaceholderResolver(placeholderResolver);
        service.setTimeout4Condition(10086);
        RagPlaceholder empty = service.ragPlaceholder();
        Assert.assertEquals(notifierManager, empty.getNotifierService());
        Assert.assertEquals(placeholderResolver, empty.getPlaceholderResolver());
        Assert.assertEquals(Integer.valueOf(10086), empty.getTimeout4Condition());
        EasyMock.verify(placeholderResolver);
    }

    @Test
    public void testAllowedNoPlaceholder() throws Exception {
        RagPlaceholder service = new RagPlaceholder();
        RagConfig config = new RagConfig();
        // 默认通过
        Assert.assertTrue(service.allowed(config, null));
    }
}
