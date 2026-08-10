package ai.open.right.workflow.flow.pubsub;

import org.junit.Assert;
import org.junit.Test;

public class PubSubQueryTest {

    @Test
    public void test() {
        PubSubQuery pubSubQuery = new PubSubQuery();
        Assert.assertFalse(pubSubQuery.hasAnswer());
        pubSubQuery.setKey("KEY");
        pubSubQuery.setQuery("QUERY");
        pubSubQuery.setAnswer("Answer");
        Assert.assertTrue(pubSubQuery.hasAnswer());
        Assert.assertEquals("KEY", pubSubQuery.getKey());
        Assert.assertEquals("QUERY", pubSubQuery.getQuery());
        Assert.assertEquals("Answer", pubSubQuery.getAnswer());
    }
}