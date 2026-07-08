package ai.deepright.llm;

import static org.springframework.util.ObjectUtils.isEmpty;

import ai.open.right.utils.DumpUtils;
import jakarta.annotation.PostConstruct;
import lombok.Getter;
import lombok.Setter;
import lombok.extern.slf4j.Slf4j;
import org.apache.commons.io.FileUtils;
import org.apache.commons.io.filefilter.FileFilterUtils;
import org.apache.commons.lang3.StringUtils;
import org.springframework.beans.BeanUtils;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.boot.autoconfigure.condition.ConditionalOnMissingBean;
import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Configuration;
import org.springframework.scheduling.annotation.Scheduled;

import java.io.File;
import java.util.Date;

@Slf4j
public class AutoDumpCleaner {

    public static final String NAME = "AutoDumpCleaner";

    protected Integer expired;

    protected String path;

    protected File dir;

    @PostConstruct
    public void init() throws Exception {
        if (!StringUtils.isEmpty(this.path)) {
            this.dir = new File(this.path);
            if (!this.dir.exists() && this.dir.mkdirs()) {
                if (log.isInfoEnabled()) {
                    log.info("The autodump cleaner dir={}", this.dir);
                }
            }
        }
    }

    @Scheduled(initialDelayString = "${autodump.llm.initialDelay:30000}", fixedRateString = "${autodump.llm.fixedRate:30000}")
    public void deletePeriod() throws Exception {
        if (this.dir != null) {
            Date cutoff = new Date(System.currentTimeMillis() - this.expired);
            for (File file : FileUtils.listFiles(this.dir, FileFilterUtils.fileFileFilter(), null)) {
                if (FileUtils.isFileOlder(file, cutoff) && StringUtils.startsWithIgnoreCase(file.getName(), DumpUtils.DUMP_PREFIX)) {
                    FileUtils.forceDelete(file);
                    if (log.isInfoEnabled()) {
                        log.info("The expired file deleted, path={}", file.getAbsolutePath());
                    }
                }
            }
        }
    }

    @Getter
    @Setter
    @Configuration
    public static class InitConfig {

        // 毫秒（30分钟）
        @Value("${autodump.llm.expired:1800000}")
        protected Integer expired;

        @Value("${autodump.llm:}")
        // LLM请求失败时自动Dump的目录
        protected String path;

        @Bean(AutoDumpCleaner.NAME)
        // 存在S3则关闭FileSys
        @ConditionalOnMissingBean(name = AutoDumpCleaner.NAME)
        public AutoDumpCleaner autoCleaner() throws Exception {
            AutoDumpCleaner autoCleaner = new AutoDumpCleaner();
            BeanUtils.copyProperties(this, autoCleaner);
            log.info("AutoDumpCleaner inited");
            return autoCleaner;
        }
    }
}
