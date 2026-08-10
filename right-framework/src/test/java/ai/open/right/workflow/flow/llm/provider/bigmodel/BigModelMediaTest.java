package ai.open.right.workflow.flow.llm.provider.bigmodel;

import ai.open.right.workflow.flow.llm.provider.openai.OpenAiMedia;
import org.junit.jupiter.api.Test;

import static org.junit.jupiter.api.Assertions.assertEquals;
import static org.junit.jupiter.api.Assertions.assertInstanceOf;
import static org.junit.jupiter.api.Assertions.assertThrows;

/**
 * {@link BigModelRequestService.BigModelMedia#getKeyUrl(String)} 及白名单校验。
 */
class BigModelMediaTest {

    @Test
    void defaultSingleton_implementsOpenAiMedia() {
        assertInstanceOf(OpenAiMedia.class, BigModelRequestService.BigModelMedia.DEFAULT);
        assertInstanceOf(BigModelRequestService.BigModelMedia.class, BigModelRequestService.BigModelMedia.DEFAULT);
    }

    @Test
    void getKeyUrl_videoMime_returnsVideoUrl() throws Exception {
        BigModelRequestService.BigModelMedia media = new BigModelRequestService.BigModelMedia();
        assertEquals("video_url", media.getKeyUrl("video/mp4"));
        assertEquals("video_url", media.getKeyUrl("video/webm"));
        assertEquals("video_url", media.getKeyUrl("video/ogg"));
        assertEquals("video_url", media.getKeyUrl("inline:video/mp4"));
    }

    @Test
    void getKeyUrl_audioMime_returnsVideoUrl() throws Exception {
        BigModelRequestService.BigModelMedia media = new BigModelRequestService.BigModelMedia();
        assertEquals("video_url", media.getKeyUrl("audio/mpeg"));
        assertEquals("video_url", media.getKeyUrl("audio/mp3"));
        assertEquals("video_url", media.getKeyUrl("audio/wav"));
    }

    @Test
    void getKeyUrl_imageMime_returnsImageUrl() throws Exception {
        BigModelRequestService.BigModelMedia media = new BigModelRequestService.BigModelMedia();
        assertEquals("image_url", media.getKeyUrl("image/png"));
        assertEquals("image_url", media.getKeyUrl("image/jpeg"));
        assertEquals("image_url", media.getKeyUrl("inline:image/bmp"));
    }

    @Test
    void getKeyUrl_pdfOrPlainText_returnsFileUrl() throws Exception {
        BigModelRequestService.BigModelMedia media = new BigModelRequestService.BigModelMedia();
        assertEquals("file_url", media.getKeyUrl("application/pdf"));
        assertEquals("file_url", media.getKeyUrl("text/plain"));
    }

    @Test
    void getKeyUrl_invalidMime_throws() {
        BigModelRequestService.BigModelMedia media = new BigModelRequestService.BigModelMedia();
        assertThrows(Exception.class, () -> media.getKeyUrl("HELLO"));
    }

    @Test
    void getPrefix_inheritedFromDefaultMedia() throws Exception {
        BigModelRequestService.BigModelMedia media = new BigModelRequestService.BigModelMedia();
        assertEquals("data:image/png;base64,", media.getPrefix("image/png"));
    }
}
