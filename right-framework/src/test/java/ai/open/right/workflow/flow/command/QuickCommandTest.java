package ai.open.right.workflow.flow.command;

import org.junit.Assert;
import org.junit.Test;

public class QuickCommandTest {

    @Test
    public void test() {
        QuickCommand command = new QuickCommand();
        command.setContent("Content");
        command.setCommand("Command");
        command.setPriority(100L);
        Assert.assertEquals(command.getCommand(), "Command");
        Assert.assertEquals(command.getContent(), "Content");
        Assert.assertEquals(command.getPriority(), Long.valueOf(100));
    }

    @Test
    public void testBuild() {
        QuickCommand command = QuickCommand.builder()
                .content("Content")
                .command("Command")
                .priority(100L)
                .build();
        Assert.assertEquals(command.getCommand(), "Command");
        Assert.assertEquals(command.getContent(), "Content");
        Assert.assertEquals(command.getPriority(), Long.valueOf(100));
    }
}
