package ai.open.right.workflow.flow.llm.provider.coze;

import ai.open.right.ObjectBuilder;
import ai.open.right.workflow.flow.llm.LLMQuery;
import ai.open.right.workflow.flow.llm.config.LLMConfig;
import ai.open.right.workflow.flow.llm.provider.ProviderRequestRewriter;
import ai.open.right.workflow.flow.llm.provider.ProviderToken;
import org.junit.Assert;
import org.junit.Test;

import java.util.HashMap;
import java.util.Map;

public class CozeRequestServiceTest {

    @Test
    public void testBuild() throws Exception {
        CozeRequestService s = new CozeRequestService();
        Assert.assertEquals(s.build().getClass(), CozeRequest.class);
    }

    @Test
    public void testConfig() throws Exception {
        CozeRequestService s = new CozeRequestService() {

            protected void buildPrompt(CozeRequest request, LLMConfig llmConfig, LLMQuery llmQuery) throws Exception {

            }

            protected void internalHistory(CozeRequest request) throws Exception {

            }
        };
        s.setProviderToken(new ProviderToken());
        s.setProviderRequestRewriter(new ProviderRequestRewriter.BaseRequestRewriter());
        LLMConfig config = new LLMConfig();
        Map<String, Object> add = new HashMap<String, Object>();
        add.put(CozeRequestService.KEY_TOKEN, "TOKEN");
        add.put(CozeRequestService.KEY_BOT, "BOT");
        config.setAdditional(add);
        CozeRequest req = s.config(config, ObjectBuilder.buildLLMQuery());
        Assert.assertNotNull(req);
    }

    @Test
    public void testHashCode1() throws Exception {
        Object object = CozeRequestService.class.getConstructor(null).newInstance(null);
        Assert.assertEquals(object.hashCode(), object.hashCode());
        Assert.assertEquals(object, object);
    }

    @Test
    public void testHashCode2() throws Exception {
        Object object = CozeRequestService.InitConfig.class.getConstructor(null).newInstance(null);
        Assert.assertEquals(object.hashCode(), object.hashCode());
        Assert.assertEquals(object, object);
    }

    @Test
    public void testGetModel() throws Exception {
        CozeRequestService service = new CozeRequestService();
        Assert.assertEquals(CozeRequestService.MODEL, service.getModel(ObjectBuilder.buildWorkflowTask()));
    }
}
