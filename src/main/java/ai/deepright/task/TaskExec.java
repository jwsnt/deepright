package ai.deepright.task;

import ai.deepright.cli.CliPubData;
import ai.deepright.cli.CliPubSub;
import ai.deepright.cli.CliSubFetcher;
import ai.deepright.cli.CliSubOps;
import ai.deepright.feature.FeatureUtils;
import ai.open.right.WorkflowException;
import ai.open.right.workflow.flow.WorkflowTask;
import ai.open.right.workflow.flow.file.DefStore;
import lombok.Builder;
import lombok.Getter;
import lombok.Setter;
import org.apache.commons.lang3.StringUtils;
import org.springframework.util.Assert;

import java.time.Instant;
import java.time.LocalDateTime;
import java.time.ZoneId;
import java.time.format.DateTimeFormatter;
import java.util.List;

@Getter
@Setter
@Builder
public class TaskExec implements Runnable {

    public static final String FORMAT = "yyyy_MM_dd_HH_mm_ss";

    public static final String PREFIX = "task_";

    public static final String SUFFIX = ".txt";

    public static final String TEMP = "tmp";

    protected CliSubFetcher cliSubFetcher;

    protected List<TaskSync> taskSyncs;

    protected WorkflowTask workTask;

    protected TaskResult taskResult;

    protected DefStore defStore;

    // 大于此值（字节，默认1.5M）的文件会被上传到DefStore后下发URL
    protected Integer oversize;

    protected String filename;

    protected String deadline;

    public TaskExec init(Integer timeout) throws Exception {
        long timestamp = System.currentTimeMillis() + timeout;
        DateTimeFormatter formatter = DateTimeFormatter.ofPattern(TaskExec.FORMAT);
        this.filename = FeatureUtils.buildWorkspace(this.workTask) + FeatureUtils.buildFileSeparator(this.workTask) + TaskExec.TEMP + FeatureUtils.buildFileSeparator(this.workTask) + TaskExec.PREFIX + (this.deadline = LocalDateTime.ofInstant(Instant.ofEpochMilli(timestamp), ZoneId.of(FeatureUtils.buildTimezone(this.workTask))).format(formatter)) + TaskExec.SUFFIX;
        return this;
    }

    @Override
    public void run() {
        try {
            StringBuffer answer = new StringBuffer();
            for (TaskSync eachTask : this.taskSyncs) {
                if (StringUtils.isEmpty(eachTask.getError())) {
                    answer.append(this.taskResult.buildAnswer(this.workTask, eachTask));
                } else {
                    answer.append(this.taskResult.buildError(this.workTask, eachTask));
                }
            }
            CliPubData pubData = this.cliSubFetcher.command(this.workTask, CliSubOps.builder()
                    .w(List.of(this.filename))
                    .exempted(true)
                    .build(), CliPubSub.buildPushCmd(this.workTask, this.defStore, this.oversize, answer.toString(), this.filename), "");
            Assert.isTrue(pubData.isOk(), pubData.getCmd());
        } catch (Exception e) {
            WorkflowException.dolog(e);
        }
    }
}