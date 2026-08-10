package ai.open.right.workflow.flow.llm.provider.openai;

import ai.open.right.WorkflowException;
import org.junit.jupiter.api.Test;

import static org.junit.jupiter.api.Assertions.assertEquals;
import static org.junit.jupiter.api.Assertions.assertInstanceOf;
import static org.junit.jupiter.api.Assertions.assertThrows;

/**
 * {@link OpenAiRequest.DefaultMedia} 作为 {@link OpenAiMedia} 的默认实现单测。
 */
public class DefaultMediaTest {

    @Test
    void implementsOpenAiMedia() {
        OpenAiRequest.DefaultMedia media = new OpenAiRequest.DefaultMedia();
        assertInstanceOf(OpenAiMedia.class, media);
    }

    @Test
    void getPrefix_appendsDataUrlBase64() throws Exception {
        OpenAiRequest.DefaultMedia media = new OpenAiRequest.DefaultMedia();
        assertEquals("data:image;base64,", media.getPrefix("image"));
        assertEquals("data:image/png;base64,", media.getPrefix("image/png"));
    }

    @Test
    void getKeyUrl_acceptsImageMimeTypes() throws Exception {
        OpenAiRequest.DefaultMedia media = new OpenAiRequest.DefaultMedia();
        assertEquals("image_url", media.getKeyUrl("image/jpeg"));
        assertEquals("image_url", media.getKeyUrl("image/bmp"));
        assertEquals("image_url", media.getKeyUrl("image/png"));
    }

    @Test
    void getKeyUrl_acceptsInlinePrefixedImageMime() throws Exception {
        OpenAiRequest.DefaultMedia media = new OpenAiRequest.DefaultMedia();
        assertEquals("image_url", media.getKeyUrl("inline:image/png"));
    }

    @Test
    void getKeyUrl_rejectsUnknownMime() {
        OpenAiRequest.DefaultMedia media = new OpenAiRequest.DefaultMedia();
        assertThrows(WorkflowException.class, () -> media.getKeyUrl("HELLO"));
        assertThrows(WorkflowException.class, () -> media.getKeyUrl("video/mp4"));
        assertThrows(WorkflowException.class, () -> media.getKeyUrl("audio/wav"));
    }
}
