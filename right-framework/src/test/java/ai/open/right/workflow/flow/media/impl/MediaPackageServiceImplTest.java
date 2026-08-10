package ai.open.right.workflow.flow.media.impl;

import ai.open.right.ObjectBuilder;
import ai.open.right.netty.chat.distribute.NettyRequest;
import ai.open.right.workflow.flow.media.MediaConfig;
import ai.open.right.workflow.flow.media.MediaContext;
import ai.open.right.workflow.flow.media.MediaPackage;
import ai.open.right.workflow.notify.impl.NotifierServiceImpl;
import org.junit.Assert;
import org.junit.Test;

import java.util.Arrays;
import java.util.List;

public class MediaPackageServiceImplTest {

    @Test
    public void test() throws Exception {
        MediaPackageServiceImpl mediaPackageService = new MediaPackageServiceImpl();
        mediaPackageService.setNotifierService(ObjectBuilder.buildActualNotifierManagerWithWriteBackContent("X;Y"));
        mediaPackageService.setTimeout4Llm(10000);
        mediaPackageService.setSplit(";");
        NettyRequest workflowTask = NettyRequest.class.cast(ObjectBuilder.buildWorkflowTask());
        MediaContext m1 = new MediaContext();
        m1.setData("D1");
        m1.setType("T1");
        MediaContext m2 = new MediaContext();
        m2.setData("D2");
        m2.setType("T2");
        workflowTask.setMediaContext(Arrays.asList(m1, m2));
        MediaConfig mediaConfig = new MediaConfig();
        List<MediaPackage> packages = mediaPackageService.pack(mediaConfig, workflowTask);
        Assert.assertEquals(Integer.valueOf(2), Integer.valueOf(packages.size()));
        Assert.assertEquals("X", packages.getFirst().getContent());
        Assert.assertEquals("D1", packages.getFirst().getSource());
        Assert.assertEquals("Y", packages.getLast().getContent());
        Assert.assertEquals("D2", packages.getLast().getSource());
    }

    @Test
    public void testInit() throws Exception {
        NotifierServiceImpl notifierManager = ObjectBuilder.buildActualNotifierManagerWithNothing();
        MediaPackageServiceImpl.InitConfig service = new MediaPackageServiceImpl.InitConfig();
        service.setNotifierService(notifierManager);
        service.setTimeout4Llm(1000);
        service.setSplit("_");
        MediaPackageServiceImpl empty = (MediaPackageServiceImpl) service.mediaPackageService();
        Assert.assertEquals(notifierManager, empty.getNotifierService());
        Assert.assertEquals(Integer.valueOf(1000), empty.getTimeout4Llm());
        Assert.assertEquals("_", empty.getSplit());
    }
}
