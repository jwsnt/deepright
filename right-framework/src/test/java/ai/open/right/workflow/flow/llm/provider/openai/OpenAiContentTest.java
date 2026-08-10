package ai.open.right.workflow.flow.llm.provider.openai;

import ai.open.right.utils.JsonUtils;
import ai.open.right.workflow.flow.llm.provider.ProviderFunCallRequest;
import ai.open.right.workflow.flow.llm.provider.ProviderFunCallResponse;
import ai.open.right.workflow.flow.llm.store.history.History;
import ai.open.right.workflow.flow.media.MediaContext;
import org.junit.Assert;
import org.junit.Test;

import java.util.ArrayList;
import java.util.HashMap;
import java.util.List;
import java.util.Map;

public class OpenAiContentTest {

    @Test
    public void testCallId() throws Exception {
        OpenAiRouter.OpenAiContent openAiContent = new OpenAiRouter.OpenAiContent(ProviderFunCallResponse.builder().id("HELLO").build());
        Assert.assertEquals("HELLO", openAiContent.getToolCallId());
    }

    @Test
    public void testCalls() throws Exception {
        Map<String, Object> ref = new HashMap<String, Object>();
        OpenAiRouter.OpenAiContent openAiContent = new OpenAiRouter.OpenAiContent(ProviderFunCallRequest.builder().refer(ref).args(ref).name("NAME").build());
        Assert.assertEquals("{\"function\":{\"arguments\":\"{}\",\"name\":\"NAME\"},\"index\":0,\"type\":\"function\"}", JsonUtils.write(openAiContent.getToolCalls()[0]));
    }

    @Test
    public void testContent() throws Exception {
        List<MediaContext> mediaContexts = new ArrayList<MediaContext>();
        MediaContext c1 = new MediaContext();
        c1.setData("Image");
        c1.setType("inline:image/jpeg");
        mediaContexts.add(c1);
        MediaContext c2 = new MediaContext();
        c2.setData("Image");
        c2.setType("image/jpeg");
        mediaContexts.add(c2);
        OpenAiRouter.OpenAiContent openAiContent = new OpenAiRouter.OpenAiContent(mediaContexts, OpenAiRequest.DefaultMedia.DEFAULT, "type", "QUERY",1L);
        Assert.assertEquals(3, List.class.cast(openAiContent.getContent()).size());
        Assert.assertEquals("text", Map.class.cast(List.class.cast(openAiContent.getContent()).getFirst()).get("type"));
        Assert.assertEquals("QUERY", Map.class.cast(List.class.cast(openAiContent.getContent()).getFirst()).get("text"));
        Assert.assertEquals("image_url", Map.class.cast(List.class.cast(openAiContent.getContent()).get(1)).get("type"));
        Assert.assertEquals("data:image/jpeg;base64,Image", Map.class.cast(Map.class.cast(List.class.cast(openAiContent.getContent()).get(1)).get("image_url")).get("url"));
        Assert.assertEquals("image_url", Map.class.cast(List.class.cast(openAiContent.getContent()).getLast()).get("type"));
        Assert.assertEquals("Image", Map.class.cast(Map.class.cast(List.class.cast(openAiContent.getContent()).getLast()).get("image_url")).get("url"));
    }

    @Test
    public void testHistoryUsesReasonAndCreatedOnlyConstructor() throws Exception {
        History history = new History();
        history.setCreated(8L);
        history.setRole(History.ROLE_ASSISTANT);
        history.setContent("answer");
        history.setReason("stored reason");

        OpenAiRouter.OpenAiContent historyContent = new OpenAiRouter.OpenAiContent(history);
        Assert.assertEquals("stored reason", historyContent.getReasoning());

        OpenAiRouter.OpenAiContent createdOnly = new OpenAiRouter.OpenAiContent(9L);
        Assert.assertEquals(Long.valueOf(9L), createdOnly.getCreated());
        Assert.assertNull(createdOnly.getRole());
        Assert.assertNull(createdOnly.getContent());
    }
}
