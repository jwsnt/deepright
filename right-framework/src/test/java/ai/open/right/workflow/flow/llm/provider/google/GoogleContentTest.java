package ai.open.right.workflow.flow.llm.provider.google;

import ai.open.right.workflow.flow.llm.provider.ProviderFunCallRequest;
import ai.open.right.workflow.flow.llm.provider.ProviderFunCallResponse;
import ai.open.right.workflow.flow.media.MediaContext;
import org.junit.Assert;
import org.junit.Test;

import java.util.ArrayList;
import java.util.List;

public class GoogleContentTest {

    @Test
    public void test1() throws Exception {
        GoogleRouter.GoogleMessage.GoogleContent content = new GoogleRouter.GoogleMessage.GoogleContent("DiscordConfigTest", "B", 1L);
        Assert.assertEquals("DiscordConfigTest", content.getParts().getFirst().getText());
        Assert.assertEquals("B", content.getRole());
    }

    @Test
    public void test2() throws Exception {
        GoogleRouter.GoogleMessage.GoogleContent content = new GoogleRouter.GoogleMessage.GoogleContent(ProviderFunCallRequest.builder().build());
        Assert.assertFalse(content.getParts().getFirst().getFunctionCall().isEmpty());
        Assert.assertEquals(GoogleRouter.GoogleMessage.GoogleContent.ROLE_ASSISTANT, content.getRole());
    }

    /**
     * 覆盖构造函数 GoogleContent(LLMFunCallResponse)：parts 含一个 GooglePart(llmFunCallResponse)，role 为 ROLE_ASSISTANT
     */
    @Test
    public void testGoogleContentWithLLMFunCallResponse() throws Exception {
        ProviderFunCallResponse response = ProviderFunCallResponse.builder().response("resp").name("fn").build();
        GoogleRouter.GoogleMessage.GoogleContent content = new GoogleRouter.GoogleMessage.GoogleContent(response);
        Assert.assertEquals(1, content.getParts().size());
        Assert.assertNotNull(content.getParts().get(0).getFunctionResponse());
        Assert.assertFalse(content.getParts().get(0).getFunctionResponse().isEmpty());
        Assert.assertEquals(GoogleRouter.GoogleMessage.GoogleContent.ROLE_ASSISTANT, content.getRole());
    }

    @Test
    public void test4() throws Exception {
        List<MediaContext> mediaContext = new ArrayList<MediaContext>();
        MediaContext c2 = new MediaContext();
        c2.setType("inline:image");
        c2.setData("Image");
        mediaContext.add(c2);
        MediaContext c3 = new MediaContext();
        c3.setType("image");
        c3.setData("http://1.2.3.com");
        mediaContext.add(c3);
        GoogleRouter.GoogleMessage.GoogleContent content = new GoogleRouter.GoogleMessage.GoogleContent(mediaContext, "image", "QUERY", 1L);
        Assert.assertEquals(3, content.getParts().size());
        Assert.assertEquals("Image", content.getParts().get(1).getInline().getData());
        Assert.assertEquals("http://1.2.3.com", content.getParts().get(2).getFile().getUri());
    }

    @Test
    public void testCreatedOnlyConstructor() throws Exception {
        GoogleRouter.GoogleMessage.GoogleContent content = new GoogleRouter.GoogleMessage.GoogleContent(9L);

        Assert.assertEquals(Long.valueOf(9L), content.getCreated());
        Assert.assertNull(content.getRole());
        Assert.assertTrue(content.getParts().isEmpty());
    }
}
