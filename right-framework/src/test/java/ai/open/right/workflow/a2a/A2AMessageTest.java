package ai.open.right.workflow.a2a;

import ai.open.right.ObjectBuilder;
import ai.open.right.utils.JsonUtils;
import ai.open.right.workflow.a2a.protocol.Message;
import ai.open.right.workflow.a2a.protocol.MessageRequest;
import ai.open.right.workflow.a2a.protocol.Part;
import ai.open.right.workflow.flow.WorkflowTask;
import org.apache.commons.io.IOUtils;
import org.junit.Assert;
import org.junit.Test;
import org.springframework.util.ResourceUtils;

import java.nio.charset.StandardCharsets;
import java.util.HashMap;
import java.util.Map;

public class A2AMessageTest {


    @Test(expected = IllegalArgumentException.class)
    public void testWithInvalidQuery() throws Exception {
        WorkflowTask workflowTask = ObjectBuilder.buildWorkflowTask();
        workflowTask.setQuery(IOUtils.toString(ResourceUtils.getURL("classpath:A2A_MessageSend_request.json").openStream(), StandardCharsets.UTF_8));
        A2AMessage message = new A2AMessage(workflowTask);
        Assert.assertNull(message.getMetadata("A", Map.class));
    }

    @Test(expected = IllegalArgumentException.class)
    public void testWithEmptyQuery() throws Exception {
        WorkflowTask workflowTask = ObjectBuilder.buildWorkflowTask();
        workflowTask.setQuery("");
        A2AMessage message = new A2AMessage(workflowTask);
        Assert.assertNull(message.getMetadata("A", Map.class));
    }

    @Test
    public void testWithMeta() throws Exception {
        WorkflowTask workflowTask = ObjectBuilder.buildWorkflowTask();
        workflowTask.setQuery(IOUtils.toString(ResourceUtils.getURL("classpath:A2A_MessageSend_metadata.json").openStream(), StandardCharsets.UTF_8));
        A2AMessage message = new A2AMessage(workflowTask);
        Assert.assertEquals("B", message.getMetadata("A", String.class));
    }

    @Test
    public void testWithMetaNested() throws Exception {
        WorkflowTask workflowTask = ObjectBuilder.buildWorkflowTask();
        workflowTask.setQuery(IOUtils.toString(ResourceUtils.getURL("classpath:A2A_MessageSend_metadata.json").openStream(), StandardCharsets.UTF_8));
        A2AMessage message = new A2AMessage(workflowTask);
        Assert.assertEquals(Integer.valueOf(100), message.getMetadata("C.D", Integer.class));
    }

    @Test
    public void testWithMetaObject() throws Exception {
        WorkflowTask workflowTask = ObjectBuilder.buildWorkflowTask();
        workflowTask.setQuery(IOUtils.toString(ResourceUtils.getURL("classpath:A2A_MessageSend_metadata.json").openStream(), StandardCharsets.UTF_8));
        A2AMessage message = new A2AMessage(workflowTask);
        Assert.assertEquals("tell me a joke", message.getMetadata("E.PART", Part.class).getText());
    }

    @Test
    public void testWithMetaALl() throws Exception {
        WorkflowTask workflowTask = ObjectBuilder.buildWorkflowTask();
        workflowTask.setQuery(IOUtils.toString(ResourceUtils.getURL("classpath:A2A_MessageSend_metadata.json").openStream(), StandardCharsets.UTF_8));
        A2AMessage message = new A2AMessage(workflowTask);
        Assert.assertEquals(3, message.getMetadata().size());
    }

    @Test
    public void testDataPart() throws Exception {
        WorkflowTask workflowTask = ObjectBuilder.buildWorkflowTask();
        workflowTask.setQuery(IOUtils.toString(ResourceUtils.getURL("classpath:A2A_MessageSend_part.json").openStream(), StandardCharsets.UTF_8));
        A2AMessage message = new A2AMessage(workflowTask);
        Assert.assertEquals("WORLD", message.getDataPart(1).get("HELLO"));
    }

    @Test
    public void testDataPartWithClass() throws Exception {
        WorkflowTask workflowTask = ObjectBuilder.buildWorkflowTask();
        workflowTask.setQuery(IOUtils.toString(ResourceUtils.getURL("classpath:A2A_MessageSend_part.json").openStream(), StandardCharsets.UTF_8));
        A2AMessage message = new A2AMessage(workflowTask);
        Assert.assertEquals("WORLD", message.getDataPart(1, Map.class).get("HELLO"));
        Assert.assertEquals("tell me a joke3", message.getDataPart(3, Part.class).getText());
    }

    @Test
    public void testDataPartWithFirstLast() throws Exception {
        WorkflowTask workflowTask = ObjectBuilder.buildWorkflowTask();
        workflowTask.setQuery(IOUtils.toString(ResourceUtils.getURL("classpath:A2A_MessageSend_part.json").openStream(), StandardCharsets.UTF_8));
        A2AMessage message = new A2AMessage(workflowTask);
        Assert.assertEquals("WORLD", message.getFirstDataPart(Map.class).get("HELLO"));
        Assert.assertEquals("WORLD", message.getFirstDataPart().get("HELLO"));
        Assert.assertEquals("tell me a joke3", message.getLastDataPart(Part.class).getText());
        Assert.assertEquals("tell me a joke3", message.getLastDataPart().get("text"));
    }

    @Test
    public void testDataPartWithObject() throws Exception {
        WorkflowTask workflowTask = ObjectBuilder.buildWorkflowTask();
        workflowTask.setQuery(IOUtils.toString(ResourceUtils.getURL("classpath:A2A_MessageSend_part.json").openStream(), StandardCharsets.UTF_8));
        A2AMessage message = new A2AMessage(workflowTask);
        Assert.assertEquals("WORLD", message.getObjectPart(1).getObject("HELLO"));
        Assert.assertEquals("WORLD", message.getFirstObjectPart().getObject("HELLO"));
        Assert.assertEquals("tell me a joke3", message.getLastObjectPart().getObject("text"));
        Assert.assertEquals(100, message.getFirstObjectPart().getObject("WORLD.OK"));
    }

    @Test
    public void testFilePart() throws Exception {
        WorkflowTask workflowTask = ObjectBuilder.buildWorkflowTask();
        workflowTask.setQuery(IOUtils.toString(ResourceUtils.getURL("classpath:A2A_MessageSend_part.json").openStream(), StandardCharsets.UTF_8));
        A2AMessage message = new A2AMessage(workflowTask);
        Assert.assertEquals("{\"mimeType\":\"image/png\",\"bytes\":\"iVBORw0KGgoAAAANSUhEUgAAAAUA...\",\"name\":\"input_image1.png\",\"content\":\"iVBORw0KGgoAAAANSUhEUgAAAAUA...\"}", JsonUtils.write(message.getFilePart(4)));
        Assert.assertEquals("{\"mimeType\":\"image/png\",\"bytes\":\"iVBORw0KGgoAAAANSUhEUgAAAAUA...\",\"name\":\"input_image1.png\",\"content\":\"iVBORw0KGgoAAAANSUhEUgAAAAUA...\"}", JsonUtils.write(message.getFirstFilePart()));
        Assert.assertEquals("{\"mimeType\":\"image/png\",\"bytes\":\"iVBORw0KGgoAAAANSUhEUgAAAAUA...\",\"name\":\"input_image2.png\",\"content\":\"iVBORw0KGgoAAAANSUhEUgAAAAUA...\"}", JsonUtils.write(message.getLastFilePart()));
    }

    @Test
    public void testTextPart() throws Exception {
        WorkflowTask workflowTask = ObjectBuilder.buildWorkflowTask();
        workflowTask.setQuery(IOUtils.toString(ResourceUtils.getURL("classpath:A2A_MessageSend_part.json").openStream(), StandardCharsets.UTF_8));
        A2AMessage message = new A2AMessage(workflowTask);
        Assert.assertEquals("tell me a joke2", message.getTextPart(2));
        Assert.assertEquals("tell me a joke1", message.getFirstTextPart());
        Assert.assertEquals("tell me a joke4", message.getLastTextPart());
    }

    @Test
    public void testGetSet() throws Exception {
        WorkflowTask workflowTask = ObjectBuilder.buildWorkflowTask();
        workflowTask.setQuery(IOUtils.toString(ResourceUtils.getURL("classpath:A2A_MessageSend_part.json").openStream(), StandardCharsets.UTF_8));
        A2AMessage message = new A2AMessage(workflowTask);
        Assert.assertNotNull(message.getMessage());
        Assert.assertEquals("9229e770-767c-417b-a0b0-f0741243c589", message.getMessageId());
        Assert.assertEquals("ABCDE", message.getContextId());
        Assert.assertEquals("56789", message.getTaskId());
        message.setMessage(null);
        Assert.assertNull(message.getMessage());
    }

    @Test
    public void testEmptyPart1() throws Exception {
        WorkflowTask workflowTask = ObjectBuilder.buildWorkflowTask();
        workflowTask.setQuery(IOUtils.toString(ResourceUtils.getURL("classpath:A2A_MessageSend_part_empty.json").openStream(), StandardCharsets.UTF_8));
        A2AMessage message = new A2AMessage(workflowTask);
        Assert.assertNull(message.getPart(11));
        Assert.assertNull(message.getLastPart("TEXT"));
        Assert.assertNull(message.getFirstPart("TEXT"));
    }

    @Test
    public void testEmptyPart2() throws Exception {
        WorkflowTask workflowTask = ObjectBuilder.buildWorkflowTask();
        workflowTask.setQuery(IOUtils.toString(ResourceUtils.getURL("classpath:A2A_MessageSend_part.json").openStream(), StandardCharsets.UTF_8));
        A2AMessage message = new A2AMessage(workflowTask);
        Assert.assertNull(message.getLastPart("TEXT_1"));
        Assert.assertNull(message.getFirstPart("TEXT_1"));
    }

    @Test
    public void testEmptyMeta() throws Exception {
        WorkflowTask workflowTask = ObjectBuilder.buildWorkflowTask();
        workflowTask.setQuery(IOUtils.toString(ResourceUtils.getURL("classpath:A2A_MessageSend_part_empty.json").openStream(), StandardCharsets.UTF_8));
        A2AMessage message = new A2AMessage(workflowTask);
        Assert.assertNull(message.getMetadata("A"));
        Assert.assertNull(message.getMetadata("B", String.class));
    }
    @Test(expected = IllegalArgumentException.class)
    public void testGetPartOutOfBounds() throws Exception {
        WorkflowTask workflowTask = ObjectBuilder.buildWorkflowTask();
        workflowTask.setQuery(IOUtils.toString(ResourceUtils.getURL("classpath:A2A_MessageSend_part.json").openStream(), StandardCharsets.UTF_8));
        A2AMessage message = new A2AMessage(workflowTask);
        message.getPart(100);
    }

    @Test
    public void testGetFirstPartNull() throws Exception {
        WorkflowTask workflowTask = ObjectBuilder.buildWorkflowTask();
        MessageRequest messageRequest = new MessageRequest();
        messageRequest.setMetadata(new HashMap<>());
        messageRequest.setMessage(new Message());
        workflowTask.setQuery(JsonUtils.write(messageRequest));
        A2AMessage message = new A2AMessage(workflowTask);
        Assert.assertNull(message.getFirstPart("KIND"));
    }
}
