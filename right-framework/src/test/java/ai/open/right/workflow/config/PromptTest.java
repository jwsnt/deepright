package ai.open.right.workflow.config;

import org.junit.Assert;
import org.junit.Test;

import java.util.ArrayList;
import java.util.List;

public class PromptTest {

    @Test
    public void testGetSet() {
        Prompt prompt = new Prompt("BIZ", "WORKFLOW", "CONTENT");
        Assert.assertEquals("BIZ", prompt.getBiz());
        Assert.assertEquals("WORKFLOW", prompt.getWorkflow());
        Assert.assertEquals("CONTENT", prompt.getContent());
    }
}
