package ai.open.right.workflow.flow.file.impl;

import ai.open.right.workflow.flow.WorkflowTask;
import org.apache.commons.io.FileUtils;
import org.apache.commons.io.IOUtils;
import org.easymock.EasyMockSupport;
import org.junit.jupiter.api.BeforeEach;
import org.junit.jupiter.api.Test;
import org.junit.jupiter.api.io.TempDir;

import java.io.File;
import java.io.InputStream;
import java.io.RandomAccessFile;
import java.nio.charset.StandardCharsets;
import java.nio.file.Files;
import java.nio.file.Path;

import static org.junit.jupiter.api.Assertions.*;

/**
 * SysStore 单元测试类
 */
class SysStoreTest extends EasyMockSupport {

    private SysStore sysStore;

    @TempDir
    Path tempDir;

    @BeforeEach
    void setUp() {
        sysStore = new SysStore();
    }

    /**
     * 覆盖 FileStore.name()：SysStore 返回常量 NAME。
     */
    @Test
    void testName() throws Exception {
        assertEquals(SysStore.NAME, sysStore.name());
    }

    /**
     * 覆盖 FileStore.supportNetwork()：SysStore 不支持网络，返回 false。
     */
    @Test
    void testSupportNetwork() throws Exception {
        assertFalse(sysStore.supportNetwork());
    }

    /**
     * 覆盖 FileStore.supportFilesys()：SysStore 支持磁盘，返回 true。
     */
    @Test
    void testSupportFilesys() throws Exception {
        assertTrue(sysStore.supportFilesys());
    }

    /**
     * 测试 init 方法：验证目录初始化和创建逻辑
     */
    @Test
    void testInit() throws Exception {
        // 准备测试路径
        String path = tempDir.resolve("test-init-dir").toString();
        sysStore.setPath(path);

        // 执行初始化
        sysStore.init();

        // 验证结果
        File dir = sysStore.getDir();
        assertNotNull(dir, "初始化后 dir 不应为空");
        assertTrue(dir.exists(), "目录应该被创建");
        assertTrue(dir.isDirectory(), "路径应该是目录");
        assertEquals(new File(path).getAbsolutePath(), dir.getAbsolutePath(), "路径应匹配");
    }

    /**
     * 覆盖 dir()：init 之后返回根目录 File，与 getDir() 同一引用且路径一致。
     */
    @Test
    void testDir_afterInit_returnsRootDirectory() throws Exception {
        String path = tempDir.resolve("sys-dir-method").toString();
        sysStore.setPath(path);
        sysStore.init();
        File fromDir = sysStore.dir();
        assertNotNull(fromDir);
        assertSame(sysStore.getDir(), fromDir);
        assertEquals(new File(path).getAbsolutePath(), fromDir.getAbsolutePath());
    }

    /**
     * 覆盖 dir()：直接 setDir 后 dir() 返回同一 File 引用。
     */
    @Test
    void testDir_afterSetDir_returnsSameReference() throws Exception {
        File root = tempDir.resolve("manual-root").toFile();
        sysStore.setDir(root);
        assertSame(root, sysStore.dir());
        assertSame(sysStore.getDir(), sysStore.dir());
    }

    /**
     * 测试 buildFile 方法：验证文件路径生成逻辑
     */
    @Test
    void testBuildFile() throws Exception {
        // 初始化环境
        sysStore.setPath(tempDir.toString());
        sysStore.setDeleteOnExit(true);
        sysStore.init();

        String suffix = ".test";

        // 执行方法
        File file = sysStore.buildFile(new byte[]{1, 2, 3, 4, 5}, suffix);

        // 验证结果
        assertNotNull(file, "生成的文件对象不应为空");
        assertEquals(sysStore.getDir(), file.getParentFile(), "父目录应为初始化目录");
        assertTrue(file.getName().endsWith(suffix), "文件名应以指定后缀结尾");
        // 验证文件名为 MD5(32 位十六进制) + 后缀
        String nameWithoutSuffix = file.getName().substring(0, file.getName().length() - suffix.length());
        assertEquals(32, nameWithoutSuffix.length(), "文件名主体应为 MD5 的 32 位十六进制");
        assertTrue(nameWithoutSuffix.matches("[a-f0-9]{32}"), "文件名主体应为小写十六进制");
    }

    /**
     * 测试 store 方法：验证文件写入和路径返回
     */
    @Test
    void testStore() throws Exception {
        // 初始化环境
        sysStore.setPath(tempDir.toString());
        sysStore.setOversize(10086);
        sysStore.setDeleteOnExit(true);
        sysStore.init();

        byte[] testData = "unit test content".getBytes(StandardCharsets.UTF_8);
        String suffix = ".txt";

        // 使用 EasyMock 模拟 WorkflowTask
        WorkflowTask mockTask = mock(WorkflowTask.class);

        // 激活 Mock
        replayAll();

        // 执行存储
        String resultPath = sysStore.store(testData, suffix, mockTask);

        // 验证 Mock 交互
        verifyAll();

        // 验证文件内容和路径
        assertNotNull(resultPath, "返回路径不应为空");
        File savedFile = new File(resultPath);
        assertEquals(Integer.valueOf(10086), sysStore.getOversize());
        assertTrue(savedFile.exists(), "文件应存在于磁盘上");
        assertArrayEquals(testData, FileUtils.readFileToByteArray(savedFile), "写入内容应与原始数据一致");
        assertTrue(resultPath.endsWith(suffix), "返回路径应包含正确后缀");
    }

    /**
     * 覆盖 store(byte[], suffix)：文件不存在时创建并写入，返回绝对路径
     */
    @Test
    void testStoreTwoArgs_fileNotExists_createsAndWrites() throws Exception {
        sysStore.setPath(tempDir.toString());
        sysStore.setDeleteOnExit(false);
        sysStore.init();
        byte[] data = "hello".getBytes(StandardCharsets.UTF_8);
        String suffix = ".dat";
        String path = sysStore.store(data, suffix);
        assertNotNull(path);
        assertTrue(path.endsWith(suffix));
        File f = new File(path);
        assertTrue(f.exists());
        assertArrayEquals(data, FileUtils.readFileToByteArray(f));
    }

    /**
     * 覆盖 store(byte[], suffix)：文件已存在时走 exists 分支并覆盖写入（同一内容写两次，第二次 file.exists() 为 true）
     */
    @Test
    void testStoreTwoArgs_fileExists_overwrites() throws Exception {
        sysStore.setPath(tempDir.toString());
        sysStore.setDeleteOnExit(false);
        sysStore.init();
        byte[] data = "same content".getBytes(StandardCharsets.UTF_8);
        String suffix = ".txt";
        String path1 = sysStore.store(data, suffix);
        assertTrue(new File(path1).exists());
        String path2 = sysStore.store(data, suffix);
        assertEquals(path1, path2);
        assertArrayEquals(data, FileUtils.readFileToByteArray(new File(path1)));
    }

    // ==================== access 方法测试 ====================

    /**
     * access：文件存在时返回只读 RandomAccessFile，可读回完整内容。
     */
    @Test
    void access_existingFile_returnsReadableRandomAccessFile() throws Exception {
        sysStore.setDir(tempDir.toFile());
        byte[] expected = "raf-access-content".getBytes(StandardCharsets.UTF_8);
        FileUtils.writeByteArrayToFile(new File(tempDir.toFile(), "read.bin"), expected);
        try (RandomAccessFile raf = sysStore.access("read.bin")) {
            assertEquals(0, raf.getFilePointer());
            assertEquals(expected.length, raf.length());
            byte[] buf = new byte[(int) raf.length()];
            raf.readFully(buf);
            assertArrayEquals(expected, buf);
        }
    }

    /**
     * access：文件不存在时返回 null（不抛 FileNotFoundException）。
     */
    @Test
    void access_missingFile_returnsNull() throws Exception {
        sysStore.setDir(tempDir.toFile());
        assertNull(sysStore.access("no_such_file.bin"));
    }

    /**
     * access：路径为目录时返回 null（RandomAccessFile 仅用于普通文件）。
     */
    @Test
    void access_directory_returnsNull() throws Exception {
        sysStore.setDir(tempDir.toFile());
        File subDir = new File(tempDir.toFile(), "only_dir");
        assertTrue(subDir.mkdirs());
        assertNull(sysStore.access("only_dir"));
    }

    /**
     * access：子目录下相对路径可打开（与 restore 一致支持 sub/name）。
     */
    @Test
    void access_subdirectoryFile_readable() throws Exception {
        File dir = tempDir.toFile();
        sysStore.setDir(dir);
        File sub = new File(dir, "sub");
        assertTrue(sub.mkdirs());
        byte[] expected = "nested-raf".getBytes(StandardCharsets.UTF_8);
        FileUtils.writeByteArrayToFile(new File(sub, "inner.txt"), expected);
        try (RandomAccessFile raf = sysStore.access("sub/inner.txt")) {
            byte[] buf = new byte[(int) raf.length()];
            raf.readFully(buf);
            assertArrayEquals(expected, buf);
        }
    }

    /**
     * access：传入绝对路径且文件位于 store 根目录内时，与相对路径等价可读。
     */
    @Test
    void access_absolutePathWithinStore_sameAsRelative() throws Exception {
        File dir = tempDir.toFile();
        sysStore.setDir(dir);
        byte[] expected = "abs-access".getBytes(StandardCharsets.UTF_8);
        File inner = new File(dir, "inner-abs.bin");
        FileUtils.writeByteArrayToFile(inner, expected);
        try (RandomAccessFile byAbs = sysStore.access(inner.getAbsolutePath());
             RandomAccessFile byRel = sysStore.access("inner-abs.bin")) {
            assertNotNull(byAbs);
            assertNotNull(byRel);
            assertEquals(byAbs.length(), byRel.length());
            byte[] bufA = new byte[(int) byAbs.length()];
            byte[] bufR = new byte[(int) byRel.length()];
            byAbs.seek(0);
            byRel.seek(0);
            byAbs.readFully(bufA);
            byRel.readFully(bufR);
            assertArrayEquals(expected, bufA);
            assertArrayEquals(expected, bufR);
        }
    }

    /**
     * access：绝对路径指向 store 目录外时拒绝（与 canonical 根目录校验一致）。
     */
    @Test
    void access_absolutePathOutsideStore_throws() throws Exception {
        File dir = tempDir.toFile();
        sysStore.setDir(dir);
        Path outside = tempDir.resolveSibling("sysstore-outside-access-" + System.nanoTime());
        Files.createDirectories(outside);
        File secret = outside.resolve("x.bin").toFile();
        FileUtils.writeByteArrayToFile(secret, new byte[]{1});
        assertThrows(IllegalArgumentException.class, () -> sysStore.access(secret.getAbsolutePath()));
    }

    // ==================== stream 方法测试 ====================

    /**
     * stream：文件存在时返回可读取完整内容的输入流。
     */
    @Test
    void stream_existingFile_returnsReadableInputStream() throws Exception {
        File dir = tempDir.toFile();
        sysStore.setDir(dir);
        byte[] expected = "stream-content".getBytes(StandardCharsets.UTF_8);
        FileUtils.writeByteArrayToFile(new File(dir, "read.txt"), expected);

        try (InputStream input = sysStore.stream("read.txt")) {
            assertNotNull(input);
            assertArrayEquals(expected, IOUtils.toByteArray(input));
        }
    }

    /**
     * stream：文件不存在时返回 null。
     */
    @Test
    void stream_missingFile_returnsNull() throws Exception {
        sysStore.setDir(tempDir.toFile());
        assertNull(sysStore.stream("no_such_file.txt"));
    }

    /**
     * stream：目录不是可读取的文件，返回 null。
     */
    @Test
    void stream_directory_returnsNull() throws Exception {
        File dir = tempDir.toFile();
        sysStore.setDir(dir);
        assertTrue(new File(dir, "only_dir").mkdirs());
        assertNull(sysStore.stream("only_dir"));
    }

    /**
     * stream：根目录内的绝对路径可正常读取。
     */
    @Test
    void stream_absolutePathWithinStore_returnsContent() throws Exception {
        File dir = tempDir.toFile();
        sysStore.setDir(dir);
        byte[] expected = "absolute-stream-content".getBytes(StandardCharsets.UTF_8);
        File target = new File(dir, "absolute.txt");
        FileUtils.writeByteArrayToFile(target, expected);

        try (InputStream input = sysStore.stream(target.getAbsolutePath())) {
            assertNotNull(input);
            assertArrayEquals(expected, IOUtils.toByteArray(input));
        }
    }

    /**
     * stream：相对路径穿越应被拒绝。
     */
    @Test
    void stream_pathTraversal_throwsException() throws Exception {
        sysStore.setDir(tempDir.toFile());
        assertThrows(IllegalArgumentException.class, () -> sysStore.stream("../../../outside.txt"));
    }

    /**
     * stream：根目录外的绝对路径应被拒绝。
     */
    @Test
    void stream_absolutePathOutsideStore_throws() throws Exception {
        File dir = tempDir.toFile();
        sysStore.setDir(dir);
        Path outside = tempDir.resolveSibling("sysstore-outside-stream-" + System.nanoTime());
        Files.createDirectories(outside);
        File secret = outside.resolve("secret.txt").toFile();
        FileUtils.writeByteArrayToFile(secret, new byte[]{1});

        assertThrows(IllegalArgumentException.class, () -> sysStore.stream(secret.getAbsolutePath()));
    }

    /**
     * deletePeriod：最后修改时间早于 (now - expired) 的文件会被删除。
     */
    @Test
    void deletePeriod_deletesFileOlderThanExpired() throws Exception {
        File dir = tempDir.toFile();
        File stale = new File(dir, "stale.bin");
        FileUtils.writeByteArrayToFile(stale, new byte[]{1});
        int expiredMs = 10_000;
        stale.setLastModified(System.currentTimeMillis() - expiredMs - 120_000L);

        sysStore.setDir(dir);
        sysStore.setExpired(expiredMs);
        sysStore.deletePeriod();

        assertFalse(stale.exists());
    }

    /**
     * deletePeriod：未超过 expired 的文件保留。
     */
    @Test
    void deletePeriod_keepsRecentFile() throws Exception {
        File dir = tempDir.toFile();
        File fresh = new File(dir, "fresh.bin");
        FileUtils.writeByteArrayToFile(fresh, new byte[]{2});
        fresh.setLastModified(System.currentTimeMillis());

        sysStore.setDir(dir);
        sysStore.setExpired(60_000);
        sysStore.deletePeriod();

        assertTrue(fresh.exists());
    }

    /**
     * deletePeriod：目录为空时不抛异常。
     */
    @Test
    void deletePeriod_emptyDirectory_noop() throws Exception {
        sysStore.setDir(tempDir.toFile());
        sysStore.setExpired(1_000);
        assertDoesNotThrow(() -> sysStore.deletePeriod());
    }

    /**
     * deletePeriod：仅处理 dir 下直接子文件，不递归子目录。
     */
    @Test
    void deletePeriod_doesNotDescendIntoSubdirectories() throws Exception {
        File dir = tempDir.toFile();
        File sub = new File(dir, "nested");
        assertTrue(sub.mkdirs());
        File nestedOld = new File(sub, "old.dat");
        FileUtils.writeByteArrayToFile(nestedOld, new byte[]{3});
        nestedOld.setLastModified(System.currentTimeMillis() - 3_600_000L);

        sysStore.setDir(dir);
        sysStore.setExpired(60_000);
        sysStore.deletePeriod();

        assertTrue(nestedOld.exists());
    }

    /**
     * deletePeriod：同时存在过期与未过期文件时，只删除过期文件。
     */
    @Test
    void deletePeriod_mixedOldAndNew_onlyOldRemoved() throws Exception {
        File dir = tempDir.toFile();
        File oldFile = new File(dir, "a.old");
        File newFile = new File(dir, "b.new");
        FileUtils.writeByteArrayToFile(oldFile, new byte[]{1});
        FileUtils.writeByteArrayToFile(newFile, new byte[]{2});
        oldFile.setLastModified(System.currentTimeMillis() - 3_600_000L);
        newFile.setLastModified(System.currentTimeMillis());

        sysStore.setDir(dir);
        sysStore.setExpired(60_000);
        sysStore.deletePeriod();

        assertFalse(oldFile.exists());
        assertTrue(newFile.exists());
    }

    // ==================== restore 方法测试 ====================

    /**
     * restore：文件存在时返回正确的字节内容。
     */
    @Test
    void restore_existingFile_returnsContent() throws Exception {
        File dir = tempDir.toFile();
        sysStore.setDir(dir);

        byte[] expected = "hello restore".getBytes(StandardCharsets.UTF_8);
        File target = new File(dir, "data.bin");
        FileUtils.writeByteArrayToFile(target, expected);

        byte[] result = sysStore.restore("data.bin");
        assertArrayEquals(expected, result);
    }

    /**
     * restore：文件不存在时返回空字节数组。
     */
    @Test
    void restore_nonExistentFile_returnsEmptyArray() throws Exception {
        File dir = tempDir.toFile();
        sysStore.setDir(dir);

        byte[] result = sysStore.restore("no_such_file.txt");
        assertNotNull(result);
        assertEquals(0, result.length);
    }

    /**
     * restore：空文件返回空字节数组。
     */
    @Test
    void restore_emptyFile_returnsEmptyArray() throws Exception {
        File dir = tempDir.toFile();
        sysStore.setDir(dir);

        File emptyFile = new File(dir, "empty.dat");
        FileUtils.writeByteArrayToFile(emptyFile, new byte[]{});

        byte[] result = sysStore.restore("empty.dat");
        assertNotNull(result);
        assertEquals(0, result.length);
    }

    /**
     * restore：路径穿越攻击（../）应抛出 IllegalArgumentException。
     */
    @Test
    void restore_pathTraversal_throwsException() throws Exception {
        File dir = tempDir.toFile();
        sysStore.setDir(dir);

        assertThrows(IllegalArgumentException.class, () -> sysStore.restore("../../../etc/passwd"));
    }

    /**
     * restore：使用相对路径穿越（子目录 + ../..）仍应被拦截。
     */
    @Test
    void restore_nestedPathTraversal_throwsException() throws Exception {
        File dir = tempDir.toFile();
        sysStore.setDir(dir);

        assertThrows(IllegalArgumentException.class, () -> sysStore.restore("sub/../../outside.txt"));
    }

    /**
     * restore：合法的子目录文件可以正常读取。
     */
    @Test
    void restore_subdirectoryFile_returnsContent() throws Exception {
        File dir = tempDir.toFile();
        sysStore.setDir(dir);

        File subDir = new File(dir, "sub");
        assertTrue(subDir.mkdirs());
        byte[] expected = "nested content".getBytes(StandardCharsets.UTF_8);
        FileUtils.writeByteArrayToFile(new File(subDir, "nested.txt"), expected);

        byte[] result = sysStore.restore("sub/nested.txt");
        assertArrayEquals(expected, result);
    }

    /**
     * restore：传入绝对路径且文件位于 store 根目录内时，与相对路径等价。
     */
    @Test
    void restore_absolutePathWithinStore_sameAsRelative() throws Exception {
        File dir = tempDir.toFile();
        sysStore.setDir(dir);
        byte[] expected = "abs-restore".getBytes(StandardCharsets.UTF_8);
        File f = new File(dir, "by-abs.bin");
        FileUtils.writeByteArrayToFile(f, expected);
        assertArrayEquals(expected, sysStore.restore(f.getAbsolutePath()));
        assertArrayEquals(expected, sysStore.restore("by-abs.bin"));
    }

    /**
     * restore：子目录下文件的绝对路径可读（与相对路径 sub/name 一致）。
     */
    @Test
    void restore_absolutePathToNestedFile_sameAsRelative() throws Exception {
        File dir = tempDir.toFile();
        sysStore.setDir(dir);
        File sub = new File(dir, "sub2");
        assertTrue(sub.mkdirs());
        byte[] expected = "nested-abs".getBytes(StandardCharsets.UTF_8);
        File nested = new File(sub, "n.txt");
        FileUtils.writeByteArrayToFile(nested, expected);
        assertArrayEquals(expected, sysStore.restore(nested.getAbsolutePath()));
        assertArrayEquals(expected, sysStore.restore("sub2/n.txt"));
    }

    /**
     * restore：绝对路径指向 store 目录外时拒绝。
     */
    @Test
    void restore_absolutePathOutsideStore_throws() throws Exception {
        File dir = tempDir.toFile();
        sysStore.setDir(dir);
        Path outside = tempDir.resolveSibling("sysstore-outside-restore-" + System.nanoTime());
        Files.createDirectories(outside);
        File secret = outside.resolve("secret.dat").toFile();
        FileUtils.writeByteArrayToFile(secret, new byte[]{1, 2, 3});
        assertThrows(IllegalArgumentException.class, () -> sysStore.restore(secret.getAbsolutePath()));
    }

    /**
     * restore：大文件读取验证。
     */
    @Test
    void restore_largeFile_returnsFullContent() throws Exception {
        File dir = tempDir.toFile();
        sysStore.setDir(dir);

        byte[] largeData = new byte[1024 * 100]; // 100KB
        for (int i = 0; i < largeData.length; i++) {
            largeData[i] = (byte) (i % 256);
        }
        FileUtils.writeByteArrayToFile(new File(dir, "large.bin"), largeData);

        byte[] result = sysStore.restore("large.bin");
        assertArrayEquals(largeData, result);
    }

    /**
     * restore：二进制内容（含 0x00 字节）正确读取。
     */
    @Test
    void restore_binaryContent_returnsExactBytes() throws Exception {
        File dir = tempDir.toFile();
        sysStore.setDir(dir);

        byte[] binary = new byte[]{0x00, 0x01, (byte) 0xFF, 0x7F, (byte) 0x80};
        FileUtils.writeByteArrayToFile(new File(dir, "binary.dat"), binary);

        byte[] result = sysStore.restore("binary.dat");
        assertArrayEquals(binary, result);
    }
}
