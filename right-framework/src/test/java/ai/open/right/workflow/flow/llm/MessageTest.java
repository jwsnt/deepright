package ai.open.right.workflow.flow.llm;

import ai.open.right.ObjectBuilder;
import org.junit.Test;

public class MessageTest {

    @Test
    public void testBuild() {
        Message message = Message.build(ObjectBuilder.buildLLMQuery());
        Message.MessageChecker.check(message);
    }
}
