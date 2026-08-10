package ai.open.right.workflow.notify.impl;

import ai.open.right.ObjectBuilder;
import ai.open.right.context.RedirectContext;
import ai.open.right.workflow.flow.WorkflowTask;
import ai.open.right.workflow.flow.llm.Segment;
import ai.open.right.workflow.flow.media.MediaContext;
import ai.open.right.workflow.notify.Notifier;
import org.easymock.EasyMock;
import org.junit.Assert;
import org.junit.Test;

import java.util.ArrayList;
import java.util.Collections;
import java.util.List;

import static org.mockito.ArgumentMatchers.any;
import static org.mockito.ArgumentMatchers.anyInt;
import static org.mockito.Mockito.mock;
import static org.mockito.Mockito.never;
import static org.mockito.Mockito.verify;
import static org.mockito.Mockito.when;

public class FeedbackNotifierTest {

    @Test
    public void testNotifier1() throws Exception {
        FeedbackNotifier feedbackNotifier = new FeedbackNotifier();
        feedbackNotifier.setSourceNotifier(ObjectBuilder.buildActualNotifierWithNothing());
        feedbackNotifier.setLocalhostNotifier(ObjectBuilder.buildActualNotifierWithNothing());
        feedbackNotifier.notify(ObjectBuilder.buildSegment(), RedirectContext.EMPTY, ObjectBuilder.buildWorkflowTask(), new ArrayList<>());
        Assert.assertNotNull(feedbackNotifier.getSourceNotifier());
        Assert.assertNotNull(feedbackNotifier.getLocalhostNotifier());
    }

    @Test
    public void testNotifier2() throws Exception {
        FeedbackNotifier feedbackNotifier = new FeedbackNotifier();
        feedbackNotifier.setSourceNotifier(ObjectBuilder.buildActualNotifierWithNothing());
        feedbackNotifier.setLocalhostNotifier(ObjectBuilder.buildActualNotifierWithNothing());
        feedbackNotifier.notify(ObjectBuilder.buildSegment(), RedirectContext.EMPTY, ObjectBuilder.buildWorkflowTask());
    }

    @Test
    public void testNotifier3() throws Exception {
        FeedbackNotifier feedbackNotifier = new FeedbackNotifier();
        feedbackNotifier.setSourceNotifier(ObjectBuilder.buildActualNotifierWithNothing());
        feedbackNotifier.setLocalhostNotifier(ObjectBuilder.buildActualNotifierWithNothing());
        feedbackNotifier.notify(ObjectBuilder.buildSegment(), ObjectBuilder.buildWorkflowTask(), new ArrayList<>());
    }

    @Test
    public void testNotifier4() throws Exception {
        FeedbackNotifier feedbackNotifier = new FeedbackNotifier();
        feedbackNotifier.setSourceNotifier(ObjectBuilder.buildActualNotifierWithNothing());
        feedbackNotifier.setLocalhostNotifier(ObjectBuilder.buildActualNotifierWithNothing());
        feedbackNotifier.notify(ObjectBuilder.buildSegment(), ObjectBuilder.buildWorkflowTask());
    }

    @Test
    public void testInit() throws Exception {
        Notifier n1 = EasyMock.createMock(Notifier.class);
        Notifier n2 = EasyMock.createMock(Notifier.class);
        EasyMock.replay(n1, n2);
        FeedbackNotifier.InitConfig feedbackNotifier = new FeedbackNotifier.InitConfig();
        feedbackNotifier.setLocalhostNotifier(n1);
        feedbackNotifier.setSourceNotifier(n2);
        FeedbackNotifier empty = feedbackNotifier.feedbackNotifier();
        Assert.assertEquals(n1, empty.getLocalhostNotifier());
        Assert.assertEquals(n2, empty.getSourceNotifier());
        EasyMock.verify(n1, n2);
    }

    /**
     * notify(四参)：未完成时只调用 Source，不 copy、不调用 Localhost / Endpoint。
     */
    @Test
    public void notify_fourArgs_whenNotFinished_onlySourceInvoked() throws Exception {
        FeedbackNotifier feedbackNotifier = new FeedbackNotifier();
        Segment segment = mock(Segment.class);
        when(segment.isFromFunMerge()).thenReturn(false);
        when(segment.isFinished()).thenReturn(false);
        Notifier source = mock(Notifier.class);
        Notifier localhost = mock(Notifier.class);
        Notifier endpoint = mock(Notifier.class);
        feedbackNotifier.setSourceNotifier(source);
        feedbackNotifier.setLocalhostNotifier(localhost);
        feedbackNotifier.setEndpointNotifier(endpoint);
        WorkflowTask writeBack = ObjectBuilder.buildWorkflowTask();
        List<MediaContext> media = Collections.emptyList();
        feedbackNotifier.notify(segment, RedirectContext.EMPTY, writeBack, media);
        verify(source).notify(segment, RedirectContext.EMPTY, writeBack, media);
        verify(localhost, never()).notify(any(), any(), any(), any());
        verify(endpoint, never()).notify(any(), any(), any(), any());
        verify(segment, never()).copyWithStart(anyInt());
    }

    /**
     * notify(四参)：已完成且非 FunCall 时先 Source(原 segment)，再 Localhost(copyWithStart(0))。
     */
    @Test
    public void notify_fourArgs_whenFinished_notFunCall_sourceThenLocalhost() throws Exception {
        FeedbackNotifier feedbackNotifier = new FeedbackNotifier();
        Segment segment = mock(Segment.class);
        Segment copy = mock(Segment.class);
        when(segment.isFromFunMerge()).thenReturn(false);
        when(segment.isFinished()).thenReturn(true);
        when(segment.copyWithStart(0)).thenReturn(copy);
        when(copy.isFromFunCall()).thenReturn(false);
        Notifier source = mock(Notifier.class);
        Notifier localhost = mock(Notifier.class);
        Notifier endpoint = mock(Notifier.class);
        feedbackNotifier.setSourceNotifier(source);
        feedbackNotifier.setLocalhostNotifier(localhost);
        feedbackNotifier.setEndpointNotifier(endpoint);
        WorkflowTask writeBack = ObjectBuilder.buildWorkflowTask();
        List<MediaContext> media = new ArrayList<>();
        feedbackNotifier.notify(segment, RedirectContext.EMPTY, writeBack, media);
        verify(source).notify(segment, RedirectContext.EMPTY, writeBack, media);
        verify(localhost).notify(copy, RedirectContext.EMPTY, writeBack, media);
        verify(endpoint, never()).notify(any(), any(), any(), any());
        verify(segment).copyWithStart(0);
    }

    /**
     * notify(四参)：已完成且 FunCall 时先 Source(原 segment)，再 Endpoint(copyWithStart(0))。
     */
    @Test
    public void notify_fourArgs_whenFinished_funCall_sourceThenEndpoint() throws Exception {
        FeedbackNotifier feedbackNotifier = new FeedbackNotifier();
        Segment segment = mock(Segment.class);
        Segment copy = mock(Segment.class);
        when(segment.isFromFunMerge()).thenReturn(false);
        when(segment.isFinished()).thenReturn(true);
        when(segment.isFromFunCall()).thenReturn(true);
        when(segment.copyWithStart(0)).thenReturn(copy);
        Notifier source = mock(Notifier.class);
        Notifier localhost = mock(Notifier.class);
        Notifier endpoint = mock(Notifier.class);
        feedbackNotifier.setSourceNotifier(source);
        feedbackNotifier.setLocalhostNotifier(localhost);
        feedbackNotifier.setEndpointNotifier(endpoint);
        WorkflowTask writeBack = ObjectBuilder.buildWorkflowTask();
        List<MediaContext> media = new ArrayList<>();
        feedbackNotifier.notify(segment, RedirectContext.EMPTY, writeBack, media);
        verify(source).notify(segment, RedirectContext.EMPTY, writeBack, media);
        verify(endpoint).notify(copy, RedirectContext.EMPTY, writeBack, media);
        verify(localhost, never()).notify(any(), any(), any(), any());
        verify(segment).copyWithStart(0);
    }

    /**
     * FunCall Merge：不调用 Source（父线程已推送）；未完成时 Localhost / Endpoint 也不调用。
     */
    @Test
    public void notify_fourArgs_whenFromFunMerge_notFinished_skipsSource() throws Exception {
        FeedbackNotifier feedbackNotifier = new FeedbackNotifier();
        Segment segment = mock(Segment.class);
        when(segment.isFromFunMerge()).thenReturn(true);
        when(segment.isFinished()).thenReturn(false);
        Notifier source = mock(Notifier.class);
        Notifier localhost = mock(Notifier.class);
        Notifier endpoint = mock(Notifier.class);
        feedbackNotifier.setSourceNotifier(source);
        feedbackNotifier.setLocalhostNotifier(localhost);
        feedbackNotifier.setEndpointNotifier(endpoint);
        WorkflowTask writeBack = ObjectBuilder.buildWorkflowTask();
        List<MediaContext> media = Collections.emptyList();
        feedbackNotifier.notify(segment, RedirectContext.EMPTY, writeBack, media);
        verify(source, never()).notify(any(), any(), any(), any());
        verify(localhost, never()).notify(any(), any(), any(), any());
        verify(endpoint, never()).notify(any(), any(), any(), any());
        verify(segment, never()).copyWithStart(anyInt());
    }

    /**
     * FunCall Merge + 已完成 + 非 FunCall：仍跳过 Source，仅 Localhost 收到 copy。
     */
    @Test
    public void notify_fourArgs_whenFromFunMerge_finished_notFunCall_skipsSource_localhostGetsCopy() throws Exception {
        FeedbackNotifier feedbackNotifier = new FeedbackNotifier();
        Segment segment = mock(Segment.class);
        Segment copy = mock(Segment.class);
        when(segment.isFromFunMerge()).thenReturn(true);
        when(segment.isFinished()).thenReturn(true);
        when(segment.copyWithStart(0)).thenReturn(copy);
        when(copy.isFromFunCall()).thenReturn(false);
        Notifier source = mock(Notifier.class);
        Notifier localhost = mock(Notifier.class);
        Notifier endpoint = mock(Notifier.class);
        feedbackNotifier.setSourceNotifier(source);
        feedbackNotifier.setLocalhostNotifier(localhost);
        feedbackNotifier.setEndpointNotifier(endpoint);
        WorkflowTask writeBack = ObjectBuilder.buildWorkflowTask();
        List<MediaContext> media = new ArrayList<>();
        feedbackNotifier.notify(segment, RedirectContext.EMPTY, writeBack, media);
        verify(source, never()).notify(any(), any(), any(), any());
        verify(localhost).notify(copy, RedirectContext.EMPTY, writeBack, media);
        verify(endpoint, never()).notify(any(), any(), any(), any());
    }

    /**
     * FunCall Merge + 已完成 + FunCall：跳过 Source，仅 Endpoint 收到 copy。
     */
    @Test
    public void notify_fourArgs_whenFromFunMerge_finished_funCall_skipsSource_endpointGetsCopy() throws Exception {
        FeedbackNotifier feedbackNotifier = new FeedbackNotifier();
        Segment segment = mock(Segment.class);
        Segment copy = mock(Segment.class);
        when(segment.isFromFunMerge()).thenReturn(true);
        when(segment.isFinished()).thenReturn(true);
        when(segment.isFromFunCall()).thenReturn(true);
        when(segment.copyWithStart(0)).thenReturn(copy);
        Notifier source = mock(Notifier.class);
        Notifier localhost = mock(Notifier.class);
        Notifier endpoint = mock(Notifier.class);
        feedbackNotifier.setSourceNotifier(source);
        feedbackNotifier.setLocalhostNotifier(localhost);
        feedbackNotifier.setEndpointNotifier(endpoint);
        WorkflowTask writeBack = ObjectBuilder.buildWorkflowTask();
        List<MediaContext> media = new ArrayList<>();
        feedbackNotifier.notify(segment, RedirectContext.EMPTY, writeBack, media);
        verify(source, never()).notify(any(), any(), any(), any());
        verify(endpoint).notify(copy, RedirectContext.EMPTY, writeBack, media);
        verify(localhost, never()).notify(any(), any(), any(), any());
        verify(segment).copyWithStart(0);
    }
}
