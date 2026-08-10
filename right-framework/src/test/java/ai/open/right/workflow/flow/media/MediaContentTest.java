package ai.open.right.workflow.flow.media;

import org.junit.jupiter.api.Assertions;
import org.junit.jupiter.api.Test;

import java.util.ArrayList;
import java.util.List;

public class MediaContentTest {

    @Test
    public void testGetterSetter() {
        List<MediaContext> mediaContext = new ArrayList<>();
        MediaContent mediaContent = new MediaContent();

        mediaContent.setMediaContext(mediaContext);
        mediaContent.setQuery("Query");

        Assertions.assertEquals("Query", mediaContent.getQuery());
        Assertions.assertEquals(mediaContext, mediaContent.getMediaContext());
    }

    @Test
    public void testHasMediaContext() {
        MediaContent mediaContent = new MediaContent();

        // 边界条件 1: 初始状态为 null
        Assertions.assertFalse(mediaContent.hasMediaContext());

        // 边界条件 2: 设置为 null
        mediaContent.setMediaContext(null);
        Assertions.assertFalse(mediaContent.hasMediaContext());

        // 边界条件 3: 设置为空集合
        mediaContent.setMediaContext(new ArrayList<>());
        Assertions.assertFalse(mediaContent.hasMediaContext());

        // 边界条件 4: 设置有元素的集合
        List<MediaContext> list = new ArrayList<>();
        list.add(new MediaContext());
        mediaContent.setMediaContext(list);
        Assertions.assertTrue(mediaContent.hasMediaContext());
    }

    @Test
    public void testHasQuery() {
        MediaContent mediaContent = new MediaContent();

        // 边界条件 1: 初始状态为 null
        Assertions.assertFalse(mediaContent.hasQuery());

        // 边界条件 2: 设置为 null
        mediaContent.setQuery(null);
        Assertions.assertFalse(mediaContent.hasQuery());

        // 边界条件 3: 设置为空字符串
        mediaContent.setQuery("");
        Assertions.assertFalse(mediaContent.hasQuery());

        // 边界条件 4: 设置包含空格的字符串 (StringUtils.isEmpty 对空格返回 false)
        mediaContent.setQuery(" ");
        Assertions.assertTrue(mediaContent.hasQuery());

        // 边界条件 5: 设置正常字符串
        mediaContent.setQuery("Search Content");
        Assertions.assertTrue(mediaContent.hasQuery());
    }
}