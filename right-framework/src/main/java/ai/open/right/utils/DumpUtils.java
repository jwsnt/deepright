package ai.open.right.utils;

import ai.open.right.workflow.flow.WorkflowTask;
import org.apache.commons.io.FileUtils;
import org.apache.commons.lang3.StringUtils;

import java.io.File;
import java.nio.charset.StandardCharsets;

public class DumpUtils {

    public static final String DUMP_PREFIX = "dump_";

    public static void dump(WorkflowTask workTask, String dir, String file, String body) throws Exception {
        String prefix = StringUtils.join(new String[]{workTask.getDevice(), workTask.getChat(), workTask.getConversation(), workTask.getBiz(), workTask.getWorkflow()}, "_");
        DumpUtils.dump(prefix, dir, file, body);
    }

    public static void dump(String prefix, String dir, String file, String body) throws Exception {
        FileUtils.forceMkdir(new File(dir));
        String target = StringUtils.join(new String[]{DumpUtils.DUMP_PREFIX, prefix, String.valueOf(System.currentTimeMillis()), file}, "_");
        FileUtils.write(new File(dir, target), body, StandardCharsets.UTF_8);
    }
}
