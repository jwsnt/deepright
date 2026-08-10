package ai.open.right.workflow.flow.llm.store.history.impl;

import ai.open.right.ObjectBuilder;
import ai.open.right.workflow.flow.llm.store.Dimension;
import ai.open.right.workflow.flow.llm.store.history.HistoryRewriter;
import ai.open.right.workflow.flow.llm.store.history.HistoryPair;
import org.easymock.EasyMock;
import org.junit.Assert;
import org.junit.Test;

import java.util.Arrays;

public class BaseHistoryStoreTest {

    @Test
    public void testStoreWithNotFilter() throws Exception {
        HistoryRewriter adjuster = EasyMock.createMock(HistoryRewriter.class);
        Dimension dimension = ObjectBuilder.buildDimension();
        HistoryPair historyPair = new HistoryPair();
        historyPair.setAnswer("Answer");
        historyPair.setQuery("Query");
        EasyMock.expect(adjuster.store(dimension, historyPair)).andReturn(historyPair).anyTimes();
        EasyMock.replay(adjuster);
        BaseHistoryStore baseHistoryStore = new BaseHistoryStore();
        baseHistoryStore.setHistoryRewriter(Arrays.asList(adjuster));
        historyPair = baseHistoryStore.store(dimension, historyPair);
        Assert.assertEquals("Answer", historyPair.getAnswer());
        Assert.assertEquals("Query", historyPair.getQuery());
        Assert.assertEquals(baseHistoryStore.getHistoryRewriter().getFirst(), adjuster);
        EasyMock.verify(adjuster);
    }

    @Test
    public void testStoreWithFilter() throws Exception {
        HistoryRewriter adjuster = EasyMock.createMock(HistoryRewriter.class);
        Dimension dimension = ObjectBuilder.buildDimension();
        HistoryPair historyPair = new HistoryPair();
        historyPair.setAnswer("Answer");
        historyPair.setQuery("Query");
        EasyMock.expect(adjuster.store(dimension, historyPair)).andReturn(null).anyTimes();
        EasyMock.replay(adjuster);
        BaseHistoryStore baseHistoryStore = new BaseHistoryStore();
        baseHistoryStore.setHistoryRewriter(Arrays.asList(adjuster));
        Assert.assertNull(baseHistoryStore.store(dimension, historyPair));
        EasyMock.verify(adjuster);
    }

    @Test
    public void testRestoreWithNotFilter() throws Exception {
        HistoryRewriter adjuster = EasyMock.createMock(HistoryRewriter.class);
        Dimension dimension = ObjectBuilder.buildDimension();
        HistoryPair historyPair = new HistoryPair();
        historyPair.setAnswer("Answer");
        historyPair.setQuery("Query");
        EasyMock.expect(adjuster.restore(dimension, historyPair)).andReturn(historyPair).anyTimes();
        EasyMock.replay(adjuster);
        BaseHistoryStore baseHistoryStore = new BaseHistoryStore();
        baseHistoryStore.setHistoryRewriter(Arrays.asList(adjuster));
        historyPair = baseHistoryStore.restore(dimension, historyPair);
        Assert.assertEquals("Answer", historyPair.getAnswer());
        Assert.assertEquals("Query", historyPair.getQuery());
        EasyMock.verify(adjuster);
    }

    @Test
    public void testRestoreWithFilter() throws Exception {
        HistoryRewriter adjuster = EasyMock.createMock(HistoryRewriter.class);
        Dimension dimension = ObjectBuilder.buildDimension();
        HistoryPair historyPair = new HistoryPair();
        historyPair.setAnswer("Answer");
        historyPair.setQuery("Query");
        EasyMock.expect(adjuster.restore(dimension, historyPair)).andReturn(null).anyTimes();
        EasyMock.replay(adjuster);
        BaseHistoryStore baseHistoryStore = new BaseHistoryStore();
        baseHistoryStore.setHistoryRewriter(Arrays.asList(adjuster));
        Assert.assertNull(baseHistoryStore.restore(dimension, historyPair));
        EasyMock.verify(adjuster);
    }
}
