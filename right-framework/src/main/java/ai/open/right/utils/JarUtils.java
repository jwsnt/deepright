package ai.open.right.utils;

import ai.open.right.WorkflowException;
import lombok.extern.slf4j.Slf4j;
import org.apache.commons.io.FileUtils;
import org.apache.commons.io.FilenameUtils;
import org.springframework.util.CollectionUtils;

import java.io.File;
import java.io.IOException;
import java.io.InputStream;
import java.util.Enumeration;
import java.util.Set;
import java.util.jar.JarEntry;
import java.util.jar.JarFile;

@Slf4j
public class JarUtils {

    public static final String SUFFIX = "\\.jar";

    // 解压Jar
    public static void unzip(File source, Set<String> suffix) throws Exception {
        File file = new File(source.getAbsolutePath().replaceAll(JarUtils.SUFFIX, ""));
        log.info("Unzip jar{}", file);
        FileUtils.forceMkdir(file);
        try (JarFile jarFile = new JarFile(source)) {
            Enumeration<JarEntry> entries = jarFile.entries();
            while (entries.hasMoreElements()) {
                JarEntry jarEntry = entries.nextElement();
                File entryFile = new File(file, jarEntry.getName());
                JarUtils.check(jarEntry, entryFile, file);
                if (jarEntry.isDirectory()) {
                    FileUtils.forceMkdir(entryFile);
                    if (log.isDebugEnabled()) {
                        log.debug("Create jar dir={}", entryFile);
                    }
                } else if (CollectionUtils.isEmpty(suffix) || suffix.contains(FilenameUtils.getExtension(jarEntry.getName()))) {
                    try (InputStream inputStream = jarFile.getInputStream(jarEntry)) {
                        FileUtils.copyInputStreamToFile(inputStream, entryFile);
                        if (log.isInfoEnabled()) {
                            log.info("Create jar file={}", jarEntry);
                        }
                    }
                }
            }
        }
    }

    public static void check(JarEntry jarEntry, File entry, File file) throws Exception {
        // 校验路径合法性，防止Zip Slip
        if (!entry.getCanonicalPath().startsWith(file.getCanonicalPath() + File.separator)) {
            throw new WorkflowException("Entry is outside of the target dir: " + jarEntry.getName());
        }
    }
}
