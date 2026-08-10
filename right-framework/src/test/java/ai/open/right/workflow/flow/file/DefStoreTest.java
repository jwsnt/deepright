package ai.open.right.workflow.flow.file;

import ai.open.right.workflow.flow.WorkflowTask;
import ai.open.right.workflow.flow.file.impl.S3Store;
import ai.open.right.workflow.flow.file.impl.SysStore;
import org.easymock.EasyMock;
import org.junit.jupiter.api.BeforeEach;
import org.junit.jupiter.api.Test;

import java.util.Collections;
import java.util.HashMap;
import java.util.Map;

import static org.junit.jupiter.api.Assertions.*;

/**
 * DefStore 单元测试类。
 */
class DefStoreTest {

    private DefStore defStore;
    private FileStore mockStore;

    @BeforeEach
    void setUp() {
        defStore = new DefStore();
        mockStore = EasyMock.createMock(FileStore.class);
    }

    @Test
    void testName_returnsDef() throws Exception {
        defStore.setDef(SysStore.NAME);
        defStore.setFileStore(Collections.singletonMap(SysStore.NAME, mockStore));
        assertEquals(SysStore.NAME, defStore.name());
    }

    @Test
    void testName_returnsCustomDef() throws Exception {
        String customKey = "file.store.custom";
        defStore.setDef(customKey);
        defStore.setFileStore(Collections.singletonMap(customKey, mockStore));
        assertEquals(customKey, defStore.name());
    }

    @Test
    void testStore_bytesSuffixWorkTask_delegatesToFetchStore() throws Exception {
        byte[] bytes = "data".getBytes();
        String suffix = ".txt";
        WorkflowTask task = EasyMock.createMock(WorkflowTask.class);
        defStore.setDef(SysStore.NAME);
        defStore.setFileStore(Collections.singletonMap(SysStore.NAME, mockStore));

        EasyMock.expect(mockStore.store(bytes, suffix, task)).andReturn("/path/to/file.txt");
        EasyMock.replay(mockStore, task);

        assertEquals("/path/to/file.txt", defStore.store(bytes, suffix, task));
        EasyMock.verify(mockStore);
    }

    @Test
    void testStore_bytesSuffix_delegatesToFetchStore() throws Exception {
        byte[] bytes = "data".getBytes();
        String suffix = ".json";
        defStore.setDef(SysStore.NAME);
        defStore.setFileStore(Collections.singletonMap(SysStore.NAME, mockStore));

        EasyMock.expect(mockStore.store(bytes, suffix)).andReturn("/path/to/file.json");
        EasyMock.replay(mockStore);

        assertEquals("/path/to/file.json", defStore.store(bytes, suffix));
        EasyMock.verify(mockStore);
    }

    @Test
    void testStore_bytesSuffixWorkTaskName_delegatesToFetchStoreWithName() throws Exception {
        byte[] bytes = "data".getBytes();
        String suffix = ".txt";
        WorkflowTask task = EasyMock.createMock(WorkflowTask.class);
        FileStore s3Mock = EasyMock.createMock(FileStore.class);
        Map<String, FileStore> map = new HashMap<>();
        map.put(SysStore.NAME, mockStore);
        map.put(S3Store.NAME, s3Mock);
        defStore.setDef(SysStore.NAME);
        defStore.setFileStore(map);

        EasyMock.expect(s3Mock.store(bytes, suffix, task)).andReturn("https://bucket/s3.txt");
        EasyMock.replay(mockStore, s3Mock, task);

        assertEquals("https://bucket/s3.txt", defStore.store(bytes, suffix, task, S3Store.NAME));
        EasyMock.verify(s3Mock);
    }

    @Test
    void testStore_bytesSuffixName_delegatesToFetchStoreWithName() throws Exception {
        byte[] bytes = "data".getBytes();
        String suffix = ".json";
        FileStore s3Mock = EasyMock.createMock(FileStore.class);
        Map<String, FileStore> map = new HashMap<>();
        map.put(SysStore.NAME, mockStore);
        map.put(S3Store.NAME, s3Mock);
        defStore.setDef(SysStore.NAME);
        defStore.setFileStore(map);

        EasyMock.expect(s3Mock.store(bytes, suffix)).andReturn("https://bucket/s3.json");
        EasyMock.replay(mockStore, s3Mock);

        assertEquals("https://bucket/s3.json", defStore.store(bytes, suffix, S3Store.NAME));
        EasyMock.verify(s3Mock);
    }

    @Test
    void testSupportFunction_returnsTrueWhenNameInMap() throws Exception {
        defStore.setFileStore(Collections.singletonMap(SysStore.NAME, mockStore));
        assertTrue(defStore.supportFunction(SysStore.NAME));
    }

    @Test
    void testSupportFunction_returnsFalseWhenNameNotInMap() throws Exception {
        defStore.setFileStore(Collections.singletonMap(SysStore.NAME, mockStore));
        assertFalse(defStore.supportFunction("missing.key"));
    }

    @Test
    void testSupportFunction_returnsFalseWhenMapEmpty() throws Exception {
        defStore.setFileStore(new HashMap<>());
        assertFalse(defStore.supportFunction(SysStore.NAME));
    }

    @Test
    void testSupportNetwork_delegatesToFetchStore() throws Exception {
        defStore.setDef(SysStore.NAME);
        defStore.setFileStore(Collections.singletonMap(SysStore.NAME, mockStore));
        EasyMock.expect(mockStore.supportNetwork()).andReturn(true);
        EasyMock.replay(mockStore);
        assertTrue(defStore.supportNetwork());
        EasyMock.verify(mockStore);
    }

    @Test
    void testSupportFilesys_delegatesToFetchStore() throws Exception {
        defStore.setDef(SysStore.NAME);
        defStore.setFileStore(Collections.singletonMap(SysStore.NAME, mockStore));
        EasyMock.expect(mockStore.supportFilesys()).andReturn(false);
        EasyMock.replay(mockStore);
        assertFalse(defStore.supportFilesys());
        EasyMock.verify(mockStore);
    }

    @Test
    void testFetchStore_throwsWhenDefNotInMap() {
        defStore.setDef("missing.key");
        defStore.setFileStore(new HashMap<>());

        IllegalArgumentException ex = assertThrows(IllegalArgumentException.class,
                () -> defStore.store("x".getBytes(), ".txt"));
        assertTrue(ex.getMessage().contains("missing.key"));
    }

    @Test
    void testFetchStore_throwsWhenNameNotInMap() {
        defStore.setDef(SysStore.NAME);
        defStore.setFileStore(Collections.singletonMap(SysStore.NAME, mockStore));

        assertThrows(IllegalArgumentException.class,
                () -> defStore.store("x".getBytes(), ".txt", "other.key"));
    }

    @Test
    void testFetchStore_throwsWhenMapNull() {
        defStore.setDef(SysStore.NAME);
        defStore.setFileStore(null);

        assertThrows(NullPointerException.class, () -> defStore.store("x".getBytes(), ".txt"));
    }

    @Test
    void testInitConfig_defStore() throws Exception {
        Map<String, FileStore> fileStoreMap = new HashMap<>();
        SysStore sysStore = new SysStore();
        sysStore.setPath(".");
        fileStoreMap.put(SysStore.NAME, sysStore);

        DefStore.InitConfig config = new DefStore.InitConfig();
        config.setFileStore(fileStoreMap);
        config.setDef(SysStore.NAME);

        DefStore store = config.defStore();
        assertNotNull(store);
        assertEquals(SysStore.NAME, store.getDef());
        assertEquals(fileStoreMap, store.getFileStore());
    }
}
