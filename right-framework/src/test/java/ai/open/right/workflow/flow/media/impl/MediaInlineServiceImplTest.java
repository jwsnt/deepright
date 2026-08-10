package ai.open.right.workflow.flow.media.impl;

import ai.open.right.workflow.flow.file.FileStore;
import org.easymock.EasyMock;
import org.junit.Assert;
import org.junit.Test;

public class MediaInlineServiceImplTest {

    @Test
    public void test() throws Exception {
        FileStore fileStore = EasyMock.createMock(FileStore.class);
        MediaInlineServiceImpl mediaInlineService = new MediaInlineServiceImpl();
        mediaInlineService.setFileStore(fileStore);
        mediaInlineService.init();
        Assert.assertEquals(fileStore, mediaInlineService.getFileStore());
    }

    @Test
    public void testInit() throws Exception {
        FileStore fileStore = EasyMock.createMock(FileStore.class);
        MediaInlineServiceImpl.InitConfig initConfig = new MediaInlineServiceImpl.InitConfig();
        initConfig.setFileStore(fileStore);
        MediaInlineServiceImpl mediaInlineService = MediaInlineServiceImpl.class.cast(initConfig.mediaInlineService());
        Assert.assertEquals(fileStore, mediaInlineService.getFileStore());
    }
}
