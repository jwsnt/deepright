package ai.open.right.utils;

import ai.open.right.WorkflowException;
import lombok.Getter;
import lombok.extern.slf4j.Slf4j;

import java.util.concurrent.TimeUnit;

@Getter
@Slf4j
abstract public class SpinExec {

    protected final Integer timeout;

    protected final Integer circle;

    public SpinExec(Integer timeout, Integer circle) {
        this.timeout = timeout;
        this.circle = circle;
    }

    abstract public Object doExec() throws Exception;

    public Object exec() throws Exception {
        if (this.getCircle() <= 0) {
            return null;
        }
        // 1. 确定总配额 (纳秒)
        long totalQuota = TimeUnit.MILLISECONDS.toNanos(this.getTimeout());
        long startTime = System.nanoTime();
        // 2. 计算总权重份额: 1 + 2 + ... + m = m*(m+1)/2
        double totalWeight = (this.getCircle() * (this.getCircle() + 1.0)) / 2.0;
        double unitNs = totalQuota / totalWeight;
        try {
            for (int i = 1; i <= this.getCircle(); i++) {
                // 执行业务逻辑
                long start = System.nanoTime();
                Object object = this.doExec();
                if (object != null) {
                    return object;
                }
                long duration = System.nanoTime() - start;
                // 3. 计算本轮预定的“逻辑时间点”偏移量
                // 第 i 轮结束时，理论上总共应该消耗的时间 = (1+..+i) * unitNs
                double cumulativeWeight = (i * (i + 1.0)) / 2.0;
                long targetEndOffset = (long) (cumulativeWeight * unitNs);
                // 4. 计算当前实际已消耗的时间
                long actualElapsed = System.nanoTime() - startTime;
                // 5. 剩余需要睡眠的时间 = 理论截止点 - 实际已消耗点
                long sleepNs = targetEndOffset - actualElapsed;
                if (sleepNs > 0) {
                    long millis = sleepNs / 1000000;
                    int nanos = (int) (sleepNs % 1000000);
                    if (log.isDebugEnabled()) {
                        log.debug("The task will sleep {} millis and {} nanos", millis, nanos);
                    }
                    Thread.sleep(millis, nanos);
                } else if (log.isDebugEnabled()) {
                    // 如果 doExec 耗时太长，已经超过了本轮理论截止点，则不睡眠，直接进入下一轮
                    log.debug("The task execution ({} ns) exceeded allocated slot, skipping sleep", duration);
                }
                // 安全检查：如果总时间已耗尽，提前退出
                if (System.nanoTime() - startTime >= totalQuota) {
                    if (log.isDebugEnabled()) {
                        log.debug("The total time quota reached");
                    }
                    break;
                }
            }
            return null;
        } catch (Exception e) {
            throw e;
        } finally {
            if (log.isDebugEnabled()) {
                log.debug("The total exec time: {} ns", System.nanoTime() - startTime);
            }
        }
    }
}