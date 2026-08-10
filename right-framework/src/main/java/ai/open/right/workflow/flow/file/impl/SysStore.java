package ai.open.right.workflow.flow.file.impl;

import ai.open.right.workflow.flow.WorkflowTask;
import jakarta.annotation.PostConstruct;
import lombok.Getter;
import lombok.Setter;
import lombok.extern.slf4j.Slf4j;
import org.apache.commons.codec.digest.DigestUtils;
import org.apache.commons.io.FileUtils;
import org.apache.commons.io.IOUtils;
import org.apache.commons.io.filefilter.FileFilterUtils;
import org.apache.commons.lang3.StringUtils;
import org.springframework.beans.BeanUtils;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.boot.autoconfigure.condition.ConditionalOnMissingBean;
import org.springframework.boot.autoconfigure.condition.ConditionalOnProperty;
import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Configuration;
import org.springframework.scheduling.annotation.Scheduled;
import org.springframework.util.Assert;

import java.io.*;
import java.util.Date;

@Getter
@Setter
@Slf4j
public class SysStore extends SafeStore {

    public static final String NAME = "file.store.sys";

    protected Boolean deleteOnExit;

    protected Integer expired;

    protected String prefix;

    protected String path;

    protected File dir;

    @PostConstruct
    public void init() throws Exception {
        Assert.hasText(this.path, "The path can not be empty");
        this.dir = new File(this.path);
        if (!this.dir.exists() && this.dir.mkdirs()) {
            if (log.isInfoEnabled()) {
                log.info("The file sys dir={}", this.dir);
            }
        }
    }

    @Override
    public String store(byte[] bytes, String suffix, WorkflowTask workTask) throws Exception {
        this.check(bytes);
        return this.store(bytes, suffix);
    }

    @Override
    public String store(byte[] bytes, String suffix) throws Exception {
        this.check(bytes);
        File file = this.buildFile(bytes, suffix);
        if (log.isDebugEnabled()) {
            log.debug("The file location={}", file.getAbsolutePath());
        }
        if (file.exists()) {
            FileUtils.writeByteArrayToFile(file, bytes);
        } else {
            if (log.isInfoEnabled()) {
                log.info("The file created={}", file.getAbsolutePath());
            }
            FileUtils.writeByteArrayToFile(file, bytes);
        }
        return file.getAbsolutePath();
    }

    public RandomAccessFile access(String name) throws Exception {
        File file = this.buildPath(name);
        return file.exists() && !file.isDirectory() ? new RandomAccessFile(file, "r") : null;
    }

    public InputStream stream(String name) throws Exception {
        File file = this.buildPath(name);
        return file.exists() && !file.isDirectory() ? new BufferedInputStream(new FileInputStream(file)) : null;
    }

    public byte[] restore(String name) throws Exception {
        File data = this.buildPath(name);
        if (data.exists()) {
            try (InputStream input = new BufferedInputStream(new FileInputStream(data))) {
                return IOUtils.toByteArray(input);
            }
        } else {
            return new byte[]{};
        }
    }

    @Override
    public Boolean supportNetwork() throws Exception {
        return false;
    }

    @Override
    public Boolean supportFilesys() throws Exception {
        return true;
    }

    @Override
    public String name() throws Exception {
        return SysStore.NAME;
    }

    public File dir() throws Exception {
        return this.dir;
    }

    protected File buildFile(byte[] bytes, String suffix) throws Exception {
        suffix = StringUtils.defaultIfEmpty(suffix, "");
        suffix = StringUtils.startsWithIgnoreCase(suffix, ".") ? suffix : ("." + suffix);
        File file = new File(this.dir, StringUtils.defaultIfEmpty(this.prefix, "") + this.buildName(bytes, suffix));
        if (this.deleteOnExit) {
            file.deleteOnExit();
        }
        return file;
    }

    protected String buildName(byte[] bytes, String suffix) throws Exception {
        return DigestUtils.md5Hex(bytes) + suffix;
    }

    protected File buildPath(String name) throws Exception {
        File file = new File(name);
        // 如果是绝对路径则直接使用
        File data = !file.isAbsolute() ? new File(this.dir, name) : file;
        // 校验路径安全性，防止路径穿越
        Assert.isTrue(data.getCanonicalFile().toPath().startsWith(this.dir.getCanonicalFile().toPath()), "The file path is invalid: " + file);
        return data;
    }

    @Scheduled(initialDelayString = "${file.store.sys.initialDelay:30000}", fixedRateString = "${file.store.sys.fixedRate:30000}")
    public void deletePeriod() throws Exception {
        Date cutoff = new Date(System.currentTimeMillis() - this.expired);
        for (File file : FileUtils.listFiles(this.dir, FileFilterUtils.fileFileFilter(), null)) {
            if (FileUtils.isFileOlder(file, cutoff) && StringUtils.startsWithIgnoreCase(file.getName(), StringUtils.defaultIfEmpty(this.prefix, ""))) {
                FileUtils.forceDelete(file);
                if (log.isInfoEnabled()) {
                    log.info("The expired file deleted, path={}", file.getAbsolutePath());
                }
            }
        }
    }

    @ConditionalOnProperty(name = "file.store.sys.enable", havingValue = "true", matchIfMissing = true)
    @Configuration
    @Setter
    @Getter
    public static class InitConfig extends DefInitConfig {

        @Value("${file.store.sys.deleteOnExit:false}")
        protected Boolean deleteOnExit;

        // 毫秒
        @Value("${file.store.sys.expired:3600000}")
        protected Integer expired;

        @Value("${file.store.sys.prefix:sys_}")
        protected String prefix;

        @Value("${file.store.sys.path:.}")
        protected String path;

        @Bean(SysStore.NAME)
        // 存在S3则关闭FileSys
        @ConditionalOnMissingBean(name = SysStore.NAME)
        public SysStore sysStore() throws Exception {
            SysStore sysStore = new SysStore();
            BeanUtils.copyProperties(this, sysStore);
            log.info("SysStore inited: path={}", sysStore.getPath());
            return sysStore;
        }
    }
}
