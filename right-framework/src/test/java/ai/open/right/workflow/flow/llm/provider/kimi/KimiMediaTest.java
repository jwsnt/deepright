package ai.open.right.workflow.flow.llm.provider.kimi;

import ai.open.right.WorkflowException;
import ai.open.right.workflow.flow.llm.provider.openai.OpenAiMedia;
import org.junit.jupiter.api.Test;

import static org.junit.jupiter.api.Assertions.assertEquals;
import static org.junit.jupiter.api.Assertions.assertInstanceOf;
import static org.junit.jupiter.api.Assertions.assertThrows;

/**
 * {@link KimiRequestService.KimiMedia} 单测（继承 {@link ai.open.right.workflow.flow.llm.provider.openai.OpenAiRequest.DefaultMedia}）。
 */
public class KimiMediaTest {

    @Test
    void getKeyUrl_imageMime_returnsImageUrl() throws Exception {
        KimiRequestService.KimiMedia media = new KimiRequestService.KimiMedia();
        assertEquals("image_url", media.getKeyUrl("image/png"));
        assertEquals("image_url", media.getKeyUrl("inline:image/jpeg"));
    }

    @Test
    void getKeyUrl_videoMime_returnsVideoUrl() throws Exception {
        KimiRequestService.KimiMedia media = new KimiRequestService.KimiMedia();
        assertEquals("video_url", media.getKeyUrl("video/mp4"));
        assertEquals("video_url", media.getKeyUrl("video/webm"));
        assertEquals("video_url", media.getKeyUrl("video/ogg"));
        assertEquals("video_url", media.getKeyUrl("inline:video/mp4"));
    }

    @Test
    void getKeyUrl_invalidMime_throws() {
        KimiRequestService.KimiMedia media = new KimiRequestService.KimiMedia();
        assertThrows(WorkflowException.class, () -> media.getKeyUrl("HELLO"));
    }

    @Test
    void getKeyUrl_audioMime_notWhitelisted_throws() {
        KimiRequestService.KimiMedia media = new KimiRequestService.KimiMedia();
        assertThrows(WorkflowException.class, () -> media.getKeyUrl("json/wav"));
    }

    @Test
    void getPrefix_inheritedFromDefaultMedia() throws Exception {
        KimiRequestService.KimiMedia media = new KimiRequestService.KimiMedia();
        assertEquals("data:image/png;base64,", media.getPrefix("image/png"));
    }
}
