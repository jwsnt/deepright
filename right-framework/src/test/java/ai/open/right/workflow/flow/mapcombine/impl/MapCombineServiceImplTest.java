package ai.open.right.workflow.flow.mapcombine.impl;

import ai.open.right.ObjectBuilder;
import ai.open.right.context.RedirectContext;
import ai.open.right.workflow.flow.WorkflowTask;
import ai.open.right.workflow.flow.llm.Segment;
import ai.open.right.workflow.flow.llm.SegmentUsage;
import ai.open.right.workflow.flow.mapcombine.Combine;
import ai.open.right.workflow.flow.mapcombine.MapCombineConfig;
import ai.open.right.workflow.flow.mapcombine.Mapping;
import ai.open.right.workflow.flow.media.MediaContext;
import ai.open.right.workflow.notify.impl.NotifierServiceImpl;
import ai.open.right.workflow.notify.NotifierWriteBack;
import ai.open.right.workflow.sync.SyncWorkflowTask;
import org.easymock.EasyMock;
import org.junit.Assert;
import org.junit.Test;

import java.util.Arrays;
import java.util.List;
import java.util.concurrent.atomic.AtomicInteger;

public class MapCombineServiceImplTest {

    @Test
    public void testSplitWithText() throws Exception {
        MapCombineServiceImpl mapCombineService = new MapCombineServiceImpl();
        List<String> segment = mapCombineService.split("Hello\r\nWorld");
        Assert.assertEquals(Integer.valueOf(segment.size()), Integer.valueOf(2));
        Assert.assertEquals("Hello", segment.get(0));
        Assert.assertEquals("World", segment.get(1));
    }

    @Test
    public void testSplitWithJson() throws Exception {
        MapCombineServiceImpl mapCombineService = new MapCombineServiceImpl();
        List<String> segment = mapCombineService.split("[\"Hello\",\"World\"]");
        Assert.assertEquals(Integer.valueOf(segment.size()), Integer.valueOf(2));
        Assert.assertEquals("Hello", segment.get(0));
        Assert.assertEquals("World", segment.get(1));
    }

    @Test
    public void testSplitWithSource() throws Exception {
        MapCombineServiceImpl mapCombineService = new MapCombineServiceImpl();
        List<String> segment = mapCombineService.split("[\"Hello\",\"World\"");
        Assert.assertEquals(Integer.valueOf(segment.size()), Integer.valueOf(1));
        Assert.assertEquals("[\"Hello\",\"World\"", segment.get(0));
    }

    @Test
    public void testExecute() throws Exception {
        NotifierServiceImpl notifierManager = ObjectBuilder.buildActualNotifierManagerWithWriteBackDirect();
        MapCombineServiceImpl mapCombineService = new MapCombineServiceImpl();
        mapCombineService.setNotifierService(notifierManager);
        mapCombineService.setTimeout4Llm(1000);
        mapCombineService.setSegment(10);
        WorkflowTask workTask = ObjectBuilder.buildWorkflowTask();
        workTask.setQuery("Hello\r\nWorld");
        MapCombineConfig mapCombineConfig = new MapCombineConfig();
        mapCombineConfig.setCombine(new Combine());
        mapCombineConfig.setMapping(new Mapping());
        mapCombineConfig.getCombine().setDynamic("Combine");
        mapCombineConfig.getMapping().setDynamic("Map");
        mapCombineConfig.getMapping().setSplit("Split");
        mapCombineService.execute(mapCombineConfig, workTask);
    }

    @Test(expected = RuntimeException.class)
    public void testGetCombineResponseFailed() throws Exception {
        NotifierServiceImpl notifierManager = new NotifierServiceImpl() {
            public void notify(Segment segment, RedirectContext redirectContext, NotifierWriteBack notifierWriteBack) throws Exception {
                throw new RuntimeException();
            }
        };
        MapCombineServiceImpl mapCombineService = new MapCombineServiceImpl();
        mapCombineService.setNotifierService(notifierManager);
        MapCombineConfig mapCombineConfig = new MapCombineConfig();
        mapCombineConfig.setCombine(new Combine());
        mapCombineConfig.setMapping(new Mapping());
        mapCombineConfig.getCombine().setStopOnFailed(true);
        mapCombineService.getCombineResponse(mapCombineConfig, ObjectBuilder.buildWorkflowTask(), Arrays.asList("Hello", "World"));
    }

    @Test(expected = RuntimeException.class)
    public void testGetMapResponseFailed() throws Exception {
        SyncWorkflowTask syncWorkflowTask = EasyMock.createMock(SyncWorkflowTask.class);
        EasyMock.expect(syncWorkflowTask.get()).andThrow(new RuntimeException()).anyTimes();
        EasyMock.replay(syncWorkflowTask);
        MapCombineServiceImpl mapCombineService = new MapCombineServiceImpl();
        try {
            MapCombineConfig mapCombineConfig = new MapCombineConfig();
            mapCombineConfig.setCombine(new Combine());
            mapCombineConfig.setMapping(new Mapping());
            mapCombineConfig.getMapping().setStopOnFailed(true);
            mapCombineService.getMapResponse(mapCombineConfig, Arrays.asList(syncWorkflowTask));
        } finally {
            EasyMock.verify(syncWorkflowTask);
        }
    }

    @Test
    public void testGetCombineResponseFailedWithStopOnFailed() throws Exception {
        AtomicInteger count = new AtomicInteger();
        NotifierServiceImpl notifierManager = new NotifierServiceImpl() {
            public void notify(Segment segment, RedirectContext redirectContext, NotifierWriteBack notifierWriteBack, List<MediaContext> mediaContext) throws Exception {
                if (count.getAndIncrement() == 1) {
                    throw new RuntimeException();
                } else {
                    segment.setUsage(new SegmentUsage());
                    notifierWriteBack.writeBack(segment);
                }
            }
        };
        MapCombineServiceImpl mapCombineService = new MapCombineServiceImpl();
        mapCombineService.setNotifierService(notifierManager);
        mapCombineService.setTimeout4Llm(10000);
        MapCombineConfig mapCombineConfig = new MapCombineConfig();
        mapCombineConfig.setCombine(new Combine());
        mapCombineConfig.setMapping(new Mapping());
        mapCombineConfig.getCombine().setDynamic("Combine");
        mapCombineConfig.getCombine().setBatch(1);
        mapCombineConfig.getMapping().setDynamic("Map");
        mapCombineConfig.getMapping().setSplit("Split");
        mapCombineConfig.getCombine().setStopOnFailed(false);
        mapCombineConfig.getMapping().setStopOnFailed(false);
        String response = mapCombineService.getCombineResponse(mapCombineConfig, ObjectBuilder.buildWorkflowTask(), Arrays.asList("Hello", "World"));
        Assert.assertEquals("Hello", response);
    }

    @Test
    public void testGetMapResponseFailedWithStopOnFailed() throws Exception {
        SyncWorkflowTask syncWorkflowTask1 = EasyMock.createMock(SyncWorkflowTask.class);
        EasyMock.expect(syncWorkflowTask1.get()).andThrow(new RuntimeException()).anyTimes();
        EasyMock.replay(syncWorkflowTask1);
        SyncWorkflowTask syncWorkflowTask2 = EasyMock.createMock(SyncWorkflowTask.class);
        EasyMock.expect(syncWorkflowTask2.get()).andReturn("Hello World").anyTimes();
        EasyMock.replay(syncWorkflowTask2);
        MapCombineServiceImpl mapCombineService = new MapCombineServiceImpl();
        MapCombineConfig mapCombineConfig = new MapCombineConfig();
        mapCombineConfig.setCombine(new Combine());
        mapCombineConfig.setMapping(new Mapping());
        mapCombineConfig.getCombine().setDynamic("Combine");
        mapCombineConfig.getMapping().setDynamic("Map");
        mapCombineConfig.getMapping().setSplit("Split");
        mapCombineConfig.getCombine().setStopOnFailed(false);
        mapCombineConfig.getMapping().setStopOnFailed(false);
        List<String> response = mapCombineService.getMapResponse(mapCombineConfig, Arrays.asList(syncWorkflowTask1, syncWorkflowTask2));
        Assert.assertEquals("Hello World", response.getFirst());
        EasyMock.verify(syncWorkflowTask1, syncWorkflowTask2);
    }

    @Test
    public void testInit() throws Exception {
        NotifierServiceImpl notifierManager = ObjectBuilder.buildActualNotifierManagerWithNothing();
        MapCombineServiceImpl.InitConfig service = new MapCombineServiceImpl.InitConfig();
        service.setNotifierService(notifierManager);
        service.setTimeout4Llm(1000);
        service.setSegment(10);
        MapCombineServiceImpl empty = (MapCombineServiceImpl) service.mapCombineService();
        Assert.assertEquals(notifierManager, empty.getNotifierService());
        Assert.assertEquals(Integer.valueOf(1000), empty.getTimeout4Llm());
        Assert.assertEquals(Integer.valueOf(10), empty.getSegment());
    }
}
