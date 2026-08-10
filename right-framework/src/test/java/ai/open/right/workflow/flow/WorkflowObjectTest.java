package ai.open.right.workflow.flow;

import org.easymock.EasyMock;
import org.junit.jupiter.api.Assertions;
import org.junit.jupiter.api.Test;

public class WorkflowObjectTest {

    @Test
    public void testAnonymousObject() throws Exception {
        WorkflowObject object = EasyMock.createMock(WorkflowObject.class);
        EasyMock.expect(object.getObjectQuery(String.class)).andReturn("test").once();
        EasyMock.replay(object);
        Assertions.assertEquals("test", object.getObjectQuery(String.class));
        EasyMock.verify(object);
    }
}
