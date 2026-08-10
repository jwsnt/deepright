package ai.open.right.workflow.flow.llm.rag.future;

import ai.open.right.WorkflowException;
import ai.open.right.workflow.flow.llm.rag.RagConfig;
import org.easymock.EasyMock;
import org.junit.Assert;
import org.junit.Test;

import java.util.concurrent.Future;
import java.util.concurrent.TimeUnit;

public class RagAsyncTest {

    @Test
    public void testWithSuccess() throws Exception {
        Future<Void> future = EasyMock.createMock(Future.class);
        EasyMock.expect(future.get(1000, TimeUnit.MILLISECONDS)).andReturn(null).anyTimes();
        EasyMock.replay(future);
        RagAsync rag = new RagAsync(new RagConfig(), future, 1000);
        rag.run();
        Assert.assertTrue(rag.getSuccess());
        EasyMock.verify(future);
    }

    @Test(expected = WorkflowException.class)
    public void testWithFailed() throws Exception {
        Future<Void> future = EasyMock.createMock(Future.class);
        EasyMock.expect(future.get(1000, TimeUnit.MILLISECONDS)).andThrow(new WorkflowException()).anyTimes();
        EasyMock.replay(future);
        RagAsync rag = new RagAsync(new RagConfig(), future, 1000);
        try {
            rag.run();
            Assert.fail();
        } finally {
            Assert.assertFalse(rag.getSuccess());
            EasyMock.verify(future);
        }
    }
}
