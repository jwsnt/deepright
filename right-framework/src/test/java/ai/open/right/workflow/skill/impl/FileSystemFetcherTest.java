package ai.open.right.workflow.skill.impl;

import ai.open.right.ObjectBuilder;
import ai.open.right.netty.chat.distribute.NettyRequest;
import ai.open.right.release.ResourceReleaser;
import ai.open.right.resouce.ResourceService;
import ai.open.right.utils.JsonUtils;
import ai.open.right.workflow.flow.WorkflowTask;
import ai.open.right.workflow.flow.config.AllowedConfig;
import ai.open.right.workflow.skill.SkillMetadata;
import com.google.common.collect.ImmutableMap;
import org.junit.Assert;
import org.junit.Test;

import java.io.File;
import java.net.URL;
import java.nio.file.Paths;
import java.util.*;

import static org.junit.Assert.assertEquals;

public class FileSystemFetcherTest {

    @Test
    public void testTextUsage() throws Exception {
        FileSystemFetcher fileSystemFetcher = new FileSystemFetcher();
        fileSystemFetcher.setResourceService(ObjectBuilder.buildResourceService());
        fileSystemFetcher.setUsage("USAGE");
        fileSystemFetcher.setExpire(1000);
        fileSystemFetcher.setCached(true);
        fileSystemFetcher.init();
        assertEquals(fileSystemFetcher.getUsage(), "USAGE");
        assertEquals(fileSystemFetcher.getUsage(), fileSystemFetcher.fetchSkills(ObjectBuilder.buildWorkflowTask()).getUsage());
        assertEquals("USAGE", fileSystemFetcher.buildUsage());
    }

    @Test
    public void testTextUsage2() throws Exception {
        FileSystemFetcher fileSystemFetcher = new FileSystemFetcher();
        fileSystemFetcher.setResourceService(ObjectBuilder.buildResourceService());
        fileSystemFetcher.setUsage("USAGE");
        fileSystemFetcher.setExpire(1000);
        fileSystemFetcher.setCached(false);
        fileSystemFetcher.init();
        assertEquals(fileSystemFetcher.getUsage(), "USAGE");
        assertEquals(fileSystemFetcher.getUsage(), fileSystemFetcher.fetchSkills(ObjectBuilder.buildWorkflowTask()).getUsage());
        assertEquals("USAGE", fileSystemFetcher.buildUsage());
    }

    @Test
    public void testURLUsage() throws Exception {
        FileSystemFetcher fileSystemFetcher = new FileSystemFetcher();
        fileSystemFetcher.setPlaceholderResolver(ObjectBuilder.buildEmptyPlaceholderResolver());
        fileSystemFetcher.setResourceService(ObjectBuilder.buildResourceService());
        fileSystemFetcher.setUsage("classpath:skills/fullskill/Skills_Usage.json");
        fileSystemFetcher.setExpire(1000);
        fileSystemFetcher.setCached(true);
        fileSystemFetcher.init();
        assertEquals(fileSystemFetcher.getUsage(), "classpath:skills/fullskill/Skills_Usage.json");
        assertEquals("{\"HELLO\":\"WORLD\"}", fileSystemFetcher.buildUsage());
    }

    @Test
    public void testURLUsage2() throws Exception {
        FileSystemFetcher fileSystemFetcher = new FileSystemFetcher();
        fileSystemFetcher.setPlaceholderResolver(ObjectBuilder.buildEmptyPlaceholderResolver());
        fileSystemFetcher.setResourceService(ObjectBuilder.buildResourceService());
        fileSystemFetcher.setUsage("classpath:skills/fullskill/Skills_Usage.json");
        fileSystemFetcher.setExpire(1000);
        fileSystemFetcher.setCached(false);
        fileSystemFetcher.init();
        assertEquals(fileSystemFetcher.getUsage(), "classpath:skills/fullskill/Skills_Usage.json");
        assertEquals("{\"HELLO\":\"WORLD\"}", fileSystemFetcher.buildUsage());
    }

    @Test
    public void testURLUsageAndException() throws Exception {
        FileSystemFetcher fileSystemFetcher = new FileSystemFetcher();
        fileSystemFetcher.setResourceService(ObjectBuilder.buildResourceService());
        fileSystemFetcher.setUsage("classpath:skills/fullskill/Skills_Usage_2.json");
        fileSystemFetcher.setExpire(1000);
        fileSystemFetcher.setCached(true);
        fileSystemFetcher.init();
        assertEquals(fileSystemFetcher.getUsage(), "classpath:skills/fullskill/Skills_Usage_2.json");
        assertEquals("classpath:skills/fullskill/Skills_Usage_2.json", fileSystemFetcher.buildUsage());
    }

    @Test
    public void testURLUsageAndException2() throws Exception {
        FileSystemFetcher fileSystemFetcher = new FileSystemFetcher();
        fileSystemFetcher.setResourceService(ObjectBuilder.buildResourceService());
        fileSystemFetcher.setUsage("classpath:skills/fullskill/Skills_Usage_2.json");
        fileSystemFetcher.setExpire(1000);
        fileSystemFetcher.setCached(false);
        fileSystemFetcher.init();
        assertEquals(fileSystemFetcher.getUsage(), "classpath:skills/fullskill/Skills_Usage_2.json");
        assertEquals("classpath:skills/fullskill/Skills_Usage_2.json", fileSystemFetcher.buildUsage());
    }

    @Test
    public void testDefaultFile() throws Exception {
        FileSystemFetcher fileSystemFetcher = new FileSystemFetcher();
        fileSystemFetcher.setPlaceholderResolver(ObjectBuilder.buildEmptyPlaceholderResolver());
        fileSystemFetcher.setResourceService(ObjectBuilder.buildResourceService());
        fileSystemFetcher.setExpire(1000);
        fileSystemFetcher.setRelease(false);
        fileSystemFetcher.setUsage("USAGE");
        fileSystemFetcher.setCached(true);
        fileSystemFetcher.init();
        Collection<SkillMetadata> skillMetadata = fileSystemFetcher.fetchSkills(ObjectBuilder.buildWorkflowTask()).getSkills().values();
        Assert.assertTrue(fileSystemFetcher.buildPath().contains("target/test-classes/skills"));
        assertEquals(Integer.valueOf(8), Integer.valueOf(skillMetadata.size()));
    }

    @Test
    public void testDefaultFile2() throws Exception {
        FileSystemFetcher fileSystemFetcher = new FileSystemFetcher();
        fileSystemFetcher.setPlaceholderResolver(ObjectBuilder.buildEmptyPlaceholderResolver());
        fileSystemFetcher.setResourceService(ObjectBuilder.buildResourceService());
        fileSystemFetcher.setExpire(1000);
        fileSystemFetcher.setRelease(false);
        fileSystemFetcher.setUsage("USAGE");
        fileSystemFetcher.setCached(false);
        fileSystemFetcher.init();
        Collection<SkillMetadata> skillMetadata = fileSystemFetcher.fetchSkills(ObjectBuilder.buildWorkflowTask()).getSkills().values();
        Assert.assertTrue(fileSystemFetcher.buildPath().contains("target/test-classes/skills"));
        assertEquals(Integer.valueOf(8), Integer.valueOf(skillMetadata.size()));
    }

    @Test
    public void testSkills1() throws Exception {
        FileSystemFetcher fileSystemFetcher = new FileSystemFetcher();
        fileSystemFetcher.setPlaceholderResolver(ObjectBuilder.buildEmptyPlaceholderResolver());
        fileSystemFetcher.setDir(System.getProperty("user.dir") + "/src/test/resources/skills");
        fileSystemFetcher.setResourceService(ObjectBuilder.buildResourceService());
        fileSystemFetcher.setUsage("USAGE");
        fileSystemFetcher.setExpire(1000);
        fileSystemFetcher.setCached(true);
        fileSystemFetcher.init();
        Collection<SkillMetadata> skillMetadata = fileSystemFetcher.fetchSkills(ObjectBuilder.buildWorkflowTask()).getSkills().values();
        assertEquals(Integer.valueOf(8), Integer.valueOf(skillMetadata.size()));
        String expect = "[{\"description\":\"Comprehensive spreadsheet creation, editing, and analysis with support for formulas, formatting, data analysis, and visualization. When Claude needs to work with spreadsheets (.xlsx, .xlsm, .csv, .tsv, etc) for: (1) Creating new spreadsheets with formulas and formatting, (2) Reading or analyzing data, (3) Modify existing spreadsheets while preserving formulas, (4) Data analysis and visualization in spreadsheets, or (5) Recalculating formulas\",\"name\":\"xlsx\"},{\"description\":\"Comprehensive PDF manipulation toolkit for extracting text and tables, creating new PDFs, merging/splitting documents, and handling forms. When Claude needs to fill in a PDF form or programmatically process, generate, or analyze PDF documents at scale.\",\"name\":\"pdf\"},{\"description\":\"Presentation creation, editing, and analysis. When Claude needs to work with presentations (.pptx files) for: (1) Creating new presentations, (2) Modifying or editing content, (3) Working with layouts, (4) Adding comments or speaker notes, or any other presentation tasks\",\"name\":\"pptx\"},{\"description\":\"Comprehensive document creation, editing, and analysis with support for tracked changes, comments, formatting preservation, and text extraction. When Claude needs to work with professional documents (.docx files) for: (1) Creating new documents, (2) Modifying or editing content, (3) Working with tracked changes, (4) Adding comments, or any other document tasks\",\"name\":\"docx\"},{\"description\":\"empty\",\"name\":\"empty\"},{\"compatibility\":\"Designed for Claude Code (or similar products)\",\"description\":\"Extract text and tables from PDF files, fill forms, merge documents.\",\"name\":\"pdf-processing\",\"metadata\":{\"xd\":\"example-org\",\"version\":\"1.0\"},\"allowed-tools\":[\"Bash(git:*)\",\"Bash(jq:*)\",\"Read\"]},{\"description\":\"Guide for creating effective skills. This skill should be used when users want to create a new skill (or update an existing skill) that extends Claude's capabilities with specialized knowledge, workflows, or tool integrations.\",\"name\":\"skill-creator\"},{\"description\":\"生成城市天际线的高质量图像，特别是包含地标建筑和特定光影效果的场景。\",\"name\":\"image-generation-city-skyline\",\"metadata\":{\"category\":\"image-generation\"}}]";
        assertEquals(expect, JsonUtils.write(skillMetadata));
    }

    @Test
    public void testSkills1_1() throws Exception {
        FileSystemFetcher fileSystemFetcher = new FileSystemFetcher();
        fileSystemFetcher.setPlaceholderResolver(ObjectBuilder.buildEmptyPlaceholderResolver());
        fileSystemFetcher.setDir(System.getProperty("user.dir") + "/src/test/resources/skills");
        fileSystemFetcher.setResourceService(ObjectBuilder.buildResourceService());
        fileSystemFetcher.setUsage("USAGE");
        fileSystemFetcher.setExpire(1000);
        fileSystemFetcher.setCached(false);
        fileSystemFetcher.init();
        Collection<SkillMetadata> skillMetadata = fileSystemFetcher.fetchSkills(ObjectBuilder.buildWorkflowTask()).getSkills().values();
        assertEquals(Integer.valueOf(8), Integer.valueOf(skillMetadata.size()));
        String expect = "[{\"description\":\"Comprehensive spreadsheet creation, editing, and analysis with support for formulas, formatting, data analysis, and visualization. When Claude needs to work with spreadsheets (.xlsx, .xlsm, .csv, .tsv, etc) for: (1) Creating new spreadsheets with formulas and formatting, (2) Reading or analyzing data, (3) Modify existing spreadsheets while preserving formulas, (4) Data analysis and visualization in spreadsheets, or (5) Recalculating formulas\",\"name\":\"xlsx\"},{\"description\":\"Comprehensive PDF manipulation toolkit for extracting text and tables, creating new PDFs, merging/splitting documents, and handling forms. When Claude needs to fill in a PDF form or programmatically process, generate, or analyze PDF documents at scale.\",\"name\":\"pdf\"},{\"description\":\"Presentation creation, editing, and analysis. When Claude needs to work with presentations (.pptx files) for: (1) Creating new presentations, (2) Modifying or editing content, (3) Working with layouts, (4) Adding comments or speaker notes, or any other presentation tasks\",\"name\":\"pptx\"},{\"description\":\"Comprehensive document creation, editing, and analysis with support for tracked changes, comments, formatting preservation, and text extraction. When Claude needs to work with professional documents (.docx files) for: (1) Creating new documents, (2) Modifying or editing content, (3) Working with tracked changes, (4) Adding comments, or any other document tasks\",\"name\":\"docx\"},{\"description\":\"empty\",\"name\":\"empty\"},{\"compatibility\":\"Designed for Claude Code (or similar products)\",\"description\":\"Extract text and tables from PDF files, fill forms, merge documents.\",\"name\":\"pdf-processing\",\"metadata\":{\"xd\":\"example-org\",\"version\":\"1.0\"},\"allowed-tools\":[\"Bash(git:*)\",\"Bash(jq:*)\",\"Read\"]},{\"description\":\"Guide for creating effective skills. This skill should be used when users want to create a new skill (or update an existing skill) that extends Claude's capabilities with specialized knowledge, workflows, or tool integrations.\",\"name\":\"skill-creator\"},{\"description\":\"生成城市天际线的高质量图像，特别是包含地标建筑和特定光影效果的场景。\",\"name\":\"image-generation-city-skyline\",\"metadata\":{\"category\":\"image-generation\"}}]";
        assertEquals(expect, JsonUtils.write(skillMetadata));
    }

    @Test
    public void testSkills2() throws Exception {
        FileSystemFetcher fileSystemFetcher = new FileSystemFetcher();
        fileSystemFetcher.setPlaceholderResolver(ObjectBuilder.buildEmptyPlaceholderResolver());
        fileSystemFetcher.setDir(System.getProperty("user.dir") + "/src/test/resources/skills");
        fileSystemFetcher.setResourceService(ObjectBuilder.buildResourceService());
        fileSystemFetcher.setUsage("USAGE");
        fileSystemFetcher.setExpire(1000);
        fileSystemFetcher.setCached(true);
        fileSystemFetcher.init();
        List<SkillMetadata> skillMetadata = new ArrayList<>(fileSystemFetcher.fetchSkills(ObjectBuilder.buildWorkflowTask()).getSkills().values());
        assertEquals(Integer.valueOf(8), Integer.valueOf(skillMetadata.size()));
        String desc = skillMetadata.getFirst().getDescription();
        String path = skillMetadata.getFirst().getPath();
        String name = skillMetadata.getFirst().getName();
        Assert.assertTrue(path.contains("skills/document-skills/xlsx/SKILL.md"));
        assertEquals("Comprehensive spreadsheet creation, editing, and analysis with support for formulas, formatting, data analysis, and visualization. When Claude needs to work with spreadsheets (.xlsx, .xlsm, .csv, .tsv, etc) for: (1) Creating new spreadsheets with formulas and formatting, (2) Reading or analyzing data, (3) Modify existing spreadsheets while preserving formulas, (4) Data analysis and visualization in spreadsheets, or (5) Recalculating formulas", desc);
        assertEquals("xlsx", name);
    }

    @Test
    public void testSkills2_2() throws Exception {
        FileSystemFetcher fileSystemFetcher = new FileSystemFetcher();
        fileSystemFetcher.setPlaceholderResolver(ObjectBuilder.buildEmptyPlaceholderResolver());
        fileSystemFetcher.setDir(System.getProperty("user.dir") + "/src/test/resources/skills");
        fileSystemFetcher.setResourceService(ObjectBuilder.buildResourceService());
        fileSystemFetcher.setUsage("USAGE");
        fileSystemFetcher.setExpire(1000);
        fileSystemFetcher.setCached(false);
        fileSystemFetcher.init();
        List<SkillMetadata> skillMetadata = new ArrayList<>(fileSystemFetcher.fetchSkills(ObjectBuilder.buildWorkflowTask()).getSkills().values());
        assertEquals(Integer.valueOf(8), Integer.valueOf(skillMetadata.size()));
        String desc = skillMetadata.getFirst().getDescription();
        String path = skillMetadata.getFirst().getPath();
        String name = skillMetadata.getFirst().getName();
        Assert.assertTrue(path.contains("skills/document-skills/xlsx/SKILL.md"));
        assertEquals("Comprehensive spreadsheet creation, editing, and analysis with support for formulas, formatting, data analysis, and visualization. When Claude needs to work with spreadsheets (.xlsx, .xlsm, .csv, .tsv, etc) for: (1) Creating new spreadsheets with formulas and formatting, (2) Reading or analyzing data, (3) Modify existing spreadsheets while preserving formulas, (4) Data analysis and visualization in spreadsheets, or (5) Recalculating formulas", desc);
        assertEquals("xlsx", name);
    }

    @Test
    public void testSkills3() throws Exception {
        FileSystemFetcher fileSystemFetcher = new FileSystemFetcher();
        fileSystemFetcher.setPlaceholderResolver(ObjectBuilder.buildEmptyPlaceholderResolver());
        fileSystemFetcher.setDir(System.getProperty("user.dir") + "/src/test/resources/skills");
        fileSystemFetcher.setResourceService(ObjectBuilder.buildResourceService());
        fileSystemFetcher.setUsage("USAGE");
        fileSystemFetcher.setExpire(1000);
        fileSystemFetcher.setCached(false);
        fileSystemFetcher.init();
        List<SkillMetadata> skillMetadata = new ArrayList<>(fileSystemFetcher.fetchSkills(ObjectBuilder.buildWorkflowTask()).getSkills().values());
        assertEquals(Integer.valueOf(8), Integer.valueOf(skillMetadata.size()));
        String desc = skillMetadata.getFirst().getDescription();
        String path = skillMetadata.getFirst().getPath();
        String name = skillMetadata.getFirst().getName();
        Assert.assertTrue(path.contains("skills/document-skills/xlsx/SKILL.md"));
        assertEquals("Comprehensive spreadsheet creation, editing, and analysis with support for formulas, formatting, data analysis, and visualization. When Claude needs to work with spreadsheets (.xlsx, .xlsm, .csv, .tsv, etc) for: (1) Creating new spreadsheets with formulas and formatting, (2) Reading or analyzing data, (3) Modify existing spreadsheets while preserving formulas, (4) Data analysis and visualization in spreadsheets, or (5) Recalculating formulas", desc);
        assertEquals("xlsx", name);
    }

    @Test
    public void testSkills3_3() throws Exception {
        FileSystemFetcher fileSystemFetcher = new FileSystemFetcher();
        fileSystemFetcher.setPlaceholderResolver(ObjectBuilder.buildEmptyPlaceholderResolver());
        fileSystemFetcher.setDir(System.getProperty("user.dir") + "/src/test/resources/skills");
        fileSystemFetcher.setResourceService(ObjectBuilder.buildResourceService());
        fileSystemFetcher.setUsage("USAGE");
        fileSystemFetcher.setExpire(1000);
        fileSystemFetcher.setCached(true);
        fileSystemFetcher.init();
        List<SkillMetadata> skillMetadata = new ArrayList<>(fileSystemFetcher.fetchSkills(ObjectBuilder.buildWorkflowTask()).getSkills().values());
        assertEquals(Integer.valueOf(8), Integer.valueOf(skillMetadata.size()));
        String desc = skillMetadata.getFirst().getDescription();
        String path = skillMetadata.getFirst().getPath();
        String name = skillMetadata.getFirst().getName();
        Assert.assertTrue(path.contains("skills/document-skills/xlsx/SKILL.md"));
        assertEquals("Comprehensive spreadsheet creation, editing, and analysis with support for formulas, formatting, data analysis, and visualization. When Claude needs to work with spreadsheets (.xlsx, .xlsm, .csv, .tsv, etc) for: (1) Creating new spreadsheets with formulas and formatting, (2) Reading or analyzing data, (3) Modify existing spreadsheets while preserving formulas, (4) Data analysis and visualization in spreadsheets, or (5) Recalculating formulas", desc);
        assertEquals("xlsx", name);
    }

    @Test
    public void testSkills4() throws Exception {
        FileSystemFetcher fileSystemFetcher = new FileSystemFetcher();
        fileSystemFetcher.setPlaceholderResolver(ObjectBuilder.buildEmptyPlaceholderResolver());
        fileSystemFetcher.setDir(System.getProperty("user.dir") + "/src/test/resources/skills");
        fileSystemFetcher.setResourceService(ObjectBuilder.buildResourceService());
        fileSystemFetcher.setUsage("USAGE");
        fileSystemFetcher.setExpire(1000);
        fileSystemFetcher.setCached(true);
        fileSystemFetcher.init();
        List<SkillMetadata> skillMetadata = new ArrayList<>(fileSystemFetcher.fetchSkills(ObjectBuilder.buildWorkflowTask()).getSkills().values());
        assertEquals(Integer.valueOf(8), Integer.valueOf(skillMetadata.size()));
        String desc = skillMetadata.getFirst().getDescription();
        String path = skillMetadata.getFirst().getPath();
        String name = skillMetadata.getFirst().getName();
        Assert.assertTrue(path.contains("document-skills/xlsx/SKILL.md"));
        assertEquals("Comprehensive spreadsheet creation, editing, and analysis with support for formulas, formatting, data analysis, and visualization. When Claude needs to work with spreadsheets (.xlsx, .xlsm, .csv, .tsv, etc) for: (1) Creating new spreadsheets with formulas and formatting, (2) Reading or analyzing data, (3) Modify existing spreadsheets while preserving formulas, (4) Data analysis and visualization in spreadsheets, or (5) Recalculating formulas", desc);
        assertEquals("xlsx", name);
    }

    @Test
    public void testSkills4_4() throws Exception {
        FileSystemFetcher fileSystemFetcher = new FileSystemFetcher();
        fileSystemFetcher.setPlaceholderResolver(ObjectBuilder.buildEmptyPlaceholderResolver());
        fileSystemFetcher.setDir(System.getProperty("user.dir") + "/src/test/resources/skills");
        fileSystemFetcher.setResourceService(ObjectBuilder.buildResourceService());
        fileSystemFetcher.setUsage("USAGE");
        fileSystemFetcher.setExpire(1000);
        fileSystemFetcher.setCached(false);
        fileSystemFetcher.init();
        List<SkillMetadata> skillMetadata = new ArrayList<>(fileSystemFetcher.fetchSkills(ObjectBuilder.buildWorkflowTask()).getSkills().values());
        assertEquals(Integer.valueOf(8), Integer.valueOf(skillMetadata.size()));
        String desc = skillMetadata.getFirst().getDescription();
        String path = skillMetadata.getFirst().getPath();
        String name = skillMetadata.getFirst().getName();
        Assert.assertTrue(path.contains("document-skills/xlsx/SKILL.md"));
        assertEquals("Comprehensive spreadsheet creation, editing, and analysis with support for formulas, formatting, data analysis, and visualization. When Claude needs to work with spreadsheets (.xlsx, .xlsm, .csv, .tsv, etc) for: (1) Creating new spreadsheets with formulas and formatting, (2) Reading or analyzing data, (3) Modify existing spreadsheets while preserving formulas, (4) Data analysis and visualization in spreadsheets, or (5) Recalculating formulas", desc);
        assertEquals("xlsx", name);
    }

    @Test
    public void testSkills5() throws Exception {
        FileSystemFetcher fileSystemFetcher = new FileSystemFetcher();
        fileSystemFetcher.setPlaceholderResolver(ObjectBuilder.buildEmptyPlaceholderResolver());
        fileSystemFetcher.setDir(System.getProperty("user.dir") + "/src/test/resources/skills");
        fileSystemFetcher.setUsage("DESC");
        fileSystemFetcher.setResourceService(ObjectBuilder.buildResourceService());
        fileSystemFetcher.setUsage("USAGE");
        fileSystemFetcher.setExpire(1000);
        fileSystemFetcher.setCached(true);
        fileSystemFetcher.init();
        List<SkillMetadata> skillMetadata = new ArrayList<>(fileSystemFetcher.fetchSkills(ObjectBuilder.buildWorkflowTask()).getSkills().values());
        Assert.assertNotNull(fileSystemFetcher.getResourceService());
        // 去掉Bad skills
        assertEquals(Integer.valueOf(8), Integer.valueOf(skillMetadata.size()));
        String expect = "[{\"description\":\"Comprehensive spreadsheet creation, editing, and analysis with support for formulas, formatting, data analysis, and visualization. When Claude needs to work with spreadsheets (.xlsx, .xlsm, .csv, .tsv, etc) for: (1) Creating new spreadsheets with formulas and formatting, (2) Reading or analyzing data, (3) Modify existing spreadsheets while preserving formulas, (4) Data analysis and visualization in spreadsheets, or (5) Recalculating formulas\",\"name\":\"xlsx\"},{\"description\":\"Comprehensive PDF manipulation toolkit for extracting text and tables, creating new PDFs, merging/splitting documents, and handling forms. When Claude needs to fill in a PDF form or programmatically process, generate, or analyze PDF documents at scale.\",\"name\":\"pdf\"},{\"description\":\"Presentation creation, editing, and analysis. When Claude needs to work with presentations (.pptx files) for: (1) Creating new presentations, (2) Modifying or editing content, (3) Working with layouts, (4) Adding comments or speaker notes, or any other presentation tasks\",\"name\":\"pptx\"},{\"description\":\"Comprehensive document creation, editing, and analysis with support for tracked changes, comments, formatting preservation, and text extraction. When Claude needs to work with professional documents (.docx files) for: (1) Creating new documents, (2) Modifying or editing content, (3) Working with tracked changes, (4) Adding comments, or any other document tasks\",\"name\":\"docx\"},{\"description\":\"empty\",\"name\":\"empty\"},{\"compatibility\":\"Designed for Claude Code (or similar products)\",\"description\":\"Extract text and tables from PDF files, fill forms, merge documents.\",\"name\":\"pdf-processing\",\"metadata\":{\"xd\":\"example-org\",\"version\":\"1.0\"},\"allowed-tools\":[\"Bash(git:*)\",\"Bash(jq:*)\",\"Read\"]},{\"description\":\"Guide for creating effective skills. This skill should be used when users want to create a new skill (or update an existing skill) that extends Claude's capabilities with specialized knowledge, workflows, or tool integrations.\",\"name\":\"skill-creator\"},{\"description\":\"生成城市天际线的高质量图像，特别是包含地标建筑和特定光影效果的场景。\",\"name\":\"image-generation-city-skyline\",\"metadata\":{\"category\":\"image-generation\"}}]";
        assertEquals(expect, JsonUtils.write(skillMetadata));
    }

    @Test
    public void testSkills5_1() throws Exception {
        FileSystemFetcher fileSystemFetcher = new FileSystemFetcher();
        fileSystemFetcher.setPlaceholderResolver(ObjectBuilder.buildEmptyPlaceholderResolver());
        fileSystemFetcher.setDir(System.getProperty("user.dir") + "/src/test/resources/skills");
        fileSystemFetcher.setUsage("DESC");
        fileSystemFetcher.setResourceService(ObjectBuilder.buildResourceService());
        fileSystemFetcher.setUsage("USAGE");
        fileSystemFetcher.setExpire(1000);
        fileSystemFetcher.setCached(false);
        fileSystemFetcher.init();
        List<SkillMetadata> skillMetadata = new ArrayList<>(fileSystemFetcher.fetchSkills(ObjectBuilder.buildWorkflowTask()).getSkills().values());
        Assert.assertNotNull(fileSystemFetcher.getResourceService());
        // 去掉Bad skills
        assertEquals(Integer.valueOf(8), Integer.valueOf(skillMetadata.size()));
        String expect = "[{\"description\":\"Comprehensive spreadsheet creation, editing, and analysis with support for formulas, formatting, data analysis, and visualization. When Claude needs to work with spreadsheets (.xlsx, .xlsm, .csv, .tsv, etc) for: (1) Creating new spreadsheets with formulas and formatting, (2) Reading or analyzing data, (3) Modify existing spreadsheets while preserving formulas, (4) Data analysis and visualization in spreadsheets, or (5) Recalculating formulas\",\"name\":\"xlsx\"},{\"description\":\"Comprehensive PDF manipulation toolkit for extracting text and tables, creating new PDFs, merging/splitting documents, and handling forms. When Claude needs to fill in a PDF form or programmatically process, generate, or analyze PDF documents at scale.\",\"name\":\"pdf\"},{\"description\":\"Presentation creation, editing, and analysis. When Claude needs to work with presentations (.pptx files) for: (1) Creating new presentations, (2) Modifying or editing content, (3) Working with layouts, (4) Adding comments or speaker notes, or any other presentation tasks\",\"name\":\"pptx\"},{\"description\":\"Comprehensive document creation, editing, and analysis with support for tracked changes, comments, formatting preservation, and text extraction. When Claude needs to work with professional documents (.docx files) for: (1) Creating new documents, (2) Modifying or editing content, (3) Working with tracked changes, (4) Adding comments, or any other document tasks\",\"name\":\"docx\"},{\"description\":\"empty\",\"name\":\"empty\"},{\"compatibility\":\"Designed for Claude Code (or similar products)\",\"description\":\"Extract text and tables from PDF files, fill forms, merge documents.\",\"name\":\"pdf-processing\",\"metadata\":{\"xd\":\"example-org\",\"version\":\"1.0\"},\"allowed-tools\":[\"Bash(git:*)\",\"Bash(jq:*)\",\"Read\"]},{\"description\":\"Guide for creating effective skills. This skill should be used when users want to create a new skill (or update an existing skill) that extends Claude's capabilities with specialized knowledge, workflows, or tool integrations.\",\"name\":\"skill-creator\"},{\"description\":\"生成城市天际线的高质量图像，特别是包含地标建筑和特定光影效果的场景。\",\"name\":\"image-generation-city-skyline\",\"metadata\":{\"category\":\"image-generation\"}}]";
        assertEquals(expect, JsonUtils.write(skillMetadata));
    }

    @Test
    public void testSkills6() throws Exception {
        FileSystemFetcher fileSystemFetcher = new FileSystemFetcher();
        fileSystemFetcher.setPlaceholderResolver(ObjectBuilder.buildEmptyPlaceholderResolver());
        fileSystemFetcher.setDir(System.getProperty("user.dir") + "/src/test/resources/skills");
        fileSystemFetcher.setUsage("DESC");
        fileSystemFetcher.setResourceService(ObjectBuilder.buildResourceService());
        fileSystemFetcher.setUsage("USAGE");
        fileSystemFetcher.setExpire(1000);
        fileSystemFetcher.setCached(true);
        fileSystemFetcher.init();
        List<SkillMetadata> skillMetadata = new ArrayList<>(fileSystemFetcher.fetchSkills(ObjectBuilder.buildWorkflowTask()).getSkills().values());
        Assert.assertNotNull(fileSystemFetcher.getResourceService());
        // 去掉Bad skills
        assertEquals(Integer.valueOf(8), Integer.valueOf(skillMetadata.size()));
        assertEquals("pdf-processing", skillMetadata.get(5).getName());
        assertEquals("Comprehensive document creation, editing, and analysis with support for tracked changes, comments, formatting preservation, and text extraction. When Claude needs to work with professional documents (.docx files) for: (1) Creating new documents, (2) Modifying or editing content, (3) Working with tracked changes, (4) Adding comments, or any other document tasks", skillMetadata.get(3).getDescription());
        assertEquals("Designed for Claude Code (or similar products)", skillMetadata.get(5).getCompatibility());
        assertEquals("[\"Bash(git:*)\",\"Bash(jq:*)\",\"Read\"]", JsonUtils.write(skillMetadata.get(5).getAllowedTools()));
        Assert.assertTrue(skillMetadata.get(5).getPath().contains("src/test/resources/skills/fullskill/SKILL.md"));
        assertEquals("{\"xd\":\"example-org\",\"version\":\"1.0\"}", JsonUtils.write(skillMetadata.get(5).getMetadata()));
    }

    @Test
    public void testSkills6_1() throws Exception {
        FileSystemFetcher fileSystemFetcher = new FileSystemFetcher();
        fileSystemFetcher.setPlaceholderResolver(ObjectBuilder.buildEmptyPlaceholderResolver());
        fileSystemFetcher.setDir(System.getProperty("user.dir") + "/src/test/resources/skills");
        fileSystemFetcher.setUsage("DESC");
        fileSystemFetcher.setResourceService(ObjectBuilder.buildResourceService());
        fileSystemFetcher.setUsage("USAGE");
        fileSystemFetcher.setExpire(1000);
        fileSystemFetcher.setCached(false);
        fileSystemFetcher.init();
        List<SkillMetadata> skillMetadata = new ArrayList<>(fileSystemFetcher.fetchSkills(ObjectBuilder.buildWorkflowTask()).getSkills().values());
        Assert.assertNotNull(fileSystemFetcher.getResourceService());
        // 去掉Bad skills
        assertEquals(Integer.valueOf(8), Integer.valueOf(skillMetadata.size()));
        assertEquals("pdf-processing", skillMetadata.get(5).getName());
        assertEquals("Comprehensive document creation, editing, and analysis with support for tracked changes, comments, formatting preservation, and text extraction. When Claude needs to work with professional documents (.docx files) for: (1) Creating new documents, (2) Modifying or editing content, (3) Working with tracked changes, (4) Adding comments, or any other document tasks", skillMetadata.get(3).getDescription());
        assertEquals("Designed for Claude Code (or similar products)", skillMetadata.get(5).getCompatibility());
        assertEquals("[\"Bash(git:*)\",\"Bash(jq:*)\",\"Read\"]", JsonUtils.write(skillMetadata.get(5).getAllowedTools()));
        Assert.assertTrue(skillMetadata.get(5).getPath().endsWith("test/resources/skills/fullskill/SKILL.md"));
        assertEquals("{\"xd\":\"example-org\",\"version\":\"1.0\"}", JsonUtils.write(skillMetadata.get(5).getMetadata()));
    }

    @Test
    public void testInit() throws Exception {
        FileSystemFetcher.InitConfig initConfig = new FileSystemFetcher.InitConfig();
        initConfig.setPlaceholderResolver(ObjectBuilder.buildEmptyPlaceholderResolver());
        initConfig.setResourceService(ObjectBuilder.buildResourceService());
        initConfig.setName("name");
        initConfig.setDir("ABC");
        initConfig.setUsage("USAGE");
        initConfig.setExpire(1000);
        initConfig.setCached(true);
        FileSystemFetcher fileSystemFetcher = FileSystemFetcher.class.cast(initConfig.skillsFetcher());
        Assert.assertNotNull(fileSystemFetcher.getPlaceholderResolver());
        Assert.assertNotNull(fileSystemFetcher.getResourceService());
        assertEquals("name", fileSystemFetcher.getName());
        assertEquals("ABC", fileSystemFetcher.getDir());
        assertEquals("USAGE", fileSystemFetcher.getUsage());
        assertEquals(Integer.valueOf(1000), fileSystemFetcher.getExpire());
    }

    @Test
    public void testInit_2() throws Exception {
        FileSystemFetcher.InitConfig initConfig = new FileSystemFetcher.InitConfig();
        initConfig.setPlaceholderResolver(ObjectBuilder.buildEmptyPlaceholderResolver());
        initConfig.setResourceService(ObjectBuilder.buildResourceService());
        initConfig.setDir("ABC");
        initConfig.setName("name");
        initConfig.setUsage("USAGE");
        initConfig.setExpire(1000);
        initConfig.setCached(false);
        FileSystemFetcher fileSystemFetcher = FileSystemFetcher.class.cast(initConfig.skillsFetcher());
        Assert.assertNotNull(fileSystemFetcher.getPlaceholderResolver());
        Assert.assertNotNull(fileSystemFetcher.getResourceService());
        assertEquals("ABC", fileSystemFetcher.getDir());
        assertEquals("USAGE", fileSystemFetcher.getUsage());
        assertEquals(Integer.valueOf(1000), fileSystemFetcher.getExpire());
    }

    @Test
    public void testContent() throws Exception {
        String content = "name: 代码编写规范和CodeReview规范\n" + "description: 各种语言编码规范、中间件使用规范、设计架构规范和CodeReview规范";
        FileSystemFetcher.SkillVisitor skillVisitor = new FileSystemFetcher.SkillVisitor(ObjectBuilder.buildEmptyPlaceholderResolver());
        Map<String, Object> skills = skillVisitor.yaml(content);
        assertEquals("各种语言编码规范、中间件使用规范、设计架构规范和CodeReview规范", skills.get("description"));
        assertEquals("代码编写规范和CodeReview规范", skills.get("name"));
    }

    @Test
    public void testFetchSkill1() throws Exception {
        FileSystemFetcher fileSystemFetcher = new FileSystemFetcher();
        fileSystemFetcher.setDir(System.getProperty("user.dir") + "/src/test/resources/skills");
        fileSystemFetcher.setPlaceholderResolver(ObjectBuilder.buildEmptyPlaceholderResolver());
        fileSystemFetcher.setResourceService(ObjectBuilder.buildResourceService());
        fileSystemFetcher.setUsage("USAGE");
        fileSystemFetcher.setExpire(100000);
        fileSystemFetcher.setCached(true);
        fileSystemFetcher.init();
        fileSystemFetcher.fetchSkills(ObjectBuilder.buildWorkflowTask());
        String content = fileSystemFetcher.fetchResource(ObjectBuilder.buildWorkflowTask(), "skill-creator", "SKILL.md");
        assertEquals("\n" +
                "The following is the beginning of resource `SKILL.md` in skill `skill-creator`\n" +
                "# Skill Creator\n" +
                "\n" +
                "This skill provides guidance for creating effective skills.\n" +
                "\n" +
                "## About Skills\n" +
                "\n" +
                "Skills are modular, self-contained packages that extend Claude's capabilities by providing\n" +
                "specialized knowledge, workflows, and tools. Think of them as \"onboarding guides\" for specific\n" +
                "domains or tasks—they transform Claude from a general-purpose agent into a specialized agent\n" +
                "equipped with procedural knowledge that no model can fully possess.\n" +
                "\n" +
                "### What Skills Provide\n" +
                "\n" +
                "1. Specialized workflows - Multi-step procedures for specific domains\n" +
                "2. Tool integrations - Instructions for working with specific file formats or APIs\n" +
                "3. Domain expertise - Company-specific knowledge, schemas, business logic\n" +
                "4. Bundled resources - Scripts, references, and assets for complex and repetitive tasks\n" +
                "\n" +
                "### Anatomy of a Skill\n" +
                "\n" +
                "Every skill consists of a required SKILL.md file and optional bundled resources:\n" +
                "\n" +
                "```\n" +
                "skill-name/\n" +
                "├── SKILL.md (required)\n" +
                "│   ├── YAML frontmatter metadata (required)\n" +
                "│   │   ├── name: (required)\n" +
                "│   │   └── description: (required)\n" +
                "│   └── Markdown instructions (required)\n" +
                "└── Bundled Resources (optional)\n" +
                "    ├── scripts/          - Executable code (Python/Bash/etc.)\n" +
                "    ├── references/       - Documentation intended to be loaded into context as needed\n" +
                "    └── assets/           - Files used in output (templates, icons, fonts, etc.)\n" +
                "```\n" +
                "\n" +
                "#### SKILL.md (required)\n" +
                "\n" +
                "**Metadata Quality:** The `name` and `description` in YAML frontmatter determine when Claude will use the skill. Be specific about what the skill does and when to use it. Use the third-person (e.g. \"This skill should be used when...\" instead of \"Use this skill when...\").\n" +
                "\n" +
                "#### Bundled Resources (optional)\n" +
                "\n" +
                "##### Scripts (`scripts/`)\n" +
                "\n" +
                "Executable code (Python/Bash/etc.) for tasks that require deterministic reliability or are repeatedly rewritten.\n" +
                "\n" +
                "- **When to include**: When the same code is being rewritten repeatedly or deterministic reliability is needed\n" +
                "- **Example**: `scripts/rotate_pdf.py` for PDF rotation tasks\n" +
                "- **Benefits**: Token efficient, deterministic, may be executed without loading into context\n" +
                "- **Note**: Scripts may still need to be read by Claude for patching or environment-specific adjustments\n" +
                "\n" +
                "##### References (`references/`)\n" +
                "\n" +
                "Documentation and reference material intended to be loaded as needed into context to inform Claude's process and thinking.\n" +
                "\n" +
                "- **When to include**: For documentation that Claude should reference while working\n" +
                "- **Examples**: `references/finance.md` for financial schemas, `references/mnda.md` for company NDA template, `references/policies.md` for company policies, `references/api_docs.md` for API specifications\n" +
                "- **Use cases**: Database schemas, API documentation, domain knowledge, company policies, detailed workflow guides\n" +
                "- **Benefits**: Keeps SKILL.md lean, loaded only when Claude determines it's needed\n" +
                "- **Best practice**: If files are large (>10k words), include grep search patterns in SKILL.md\n" +
                "- **Avoid duplication**: Information should live in either SKILL.md or references files, not both. Prefer references files for detailed information unless it's truly core to the skill—this keeps SKILL.md lean while making information discoverable without hogging the context window. Keep only essential procedural instructions and workflow guidance in SKILL.md; move detailed reference material, schemas, and examples to references files.\n" +
                "\n" +
                "##### Assets (`assets/`)\n" +
                "\n" +
                "Files not intended to be loaded into context, but rather used within the output Claude produces.\n" +
                "\n" +
                "- **When to include**: When the skill needs files that will be used in the final output\n" +
                "- **Examples**: `assets/logo.png` for brand assets, `assets/slides.pptx` for PowerPoint templates, `assets/frontend-template/` for HTML/React boilerplate, `assets/font.ttf` for typography\n" +
                "- **Use cases**: Templates, images, icons, boilerplate code, fonts, sample documents that get copied or modified\n" +
                "- **Benefits**: Separates output resources from documentation, enables Claude to use files without loading them into context\n" +
                "\n" +
                "### Progressive Disclosure Design Principle\n" +
                "\n" +
                "Skills use a three-level loading system to manage context efficiently:\n" +
                "\n" +
                "1. **Metadata (name + description)** - Always in context (~100 words)\n" +
                "2. **SKILL.md body** - When skill triggers (<5k words)\n" +
                "3. **Bundled resources** - As needed by Claude (Unlimited*)\n" +
                "\n" +
                "*Unlimited because scripts can be executed without reading into context window.\n" +
                "\n" +
                "## Skill Creation Process\n" +
                "\n" +
                "To create a skill, follow the \"Skill Creation Process\" in order, skipping steps only if there is a clear reason why they are not applicable.\n" +
                "\n" +
                "### Step 1: Understanding the Skill with Concrete Examples\n" +
                "\n" +
                "Skip this step only when the skill's usage patterns are already clearly understood. It remains valuable even when working with an existing skill.\n" +
                "\n" +
                "To create an effective skill, clearly understand concrete examples of how the skill will be used. This understanding can come from either direct user examples or generated examples that are validated with user feedback.\n" +
                "\n" +
                "For example, when building an image-editor skill, relevant questions include:\n" +
                "\n" +
                "- \"What functionality should the image-editor skill support? Editing, rotating, anything else?\"\n" +
                "- \"Can you give some examples of how this skill would be used?\"\n" +
                "- \"I can imagine users asking for things like 'Remove the red-eye from this image' or 'Rotate this image'. Are there other ways you imagine this skill being used?\"\n" +
                "- \"What would a user say that should trigger this skill?\"\n" +
                "\n" +
                "To avoid overwhelming users, avoid asking too many questions in a single message. Start with the most important questions and follow up as needed for better effectiveness.\n" +
                "\n" +
                "Conclude this step when there is a clear sense of the functionality the skill should support.\n" +
                "\n" +
                "### Step 2: Planning the Reusable Skill Contents\n" +
                "\n" +
                "To turn concrete examples into an effective skill, analyze each example by:\n" +
                "\n" +
                "1. Considering how to execute on the example from scratch\n" +
                "2. Identifying what scripts, references, and assets would be helpful when executing these workflows repeatedly\n" +
                "\n" +
                "Example: When building a `pdf-editor` skill to handle queries like \"Help me rotate this PDF,\" the analysis shows:\n" +
                "\n" +
                "1. Rotating a PDF requires re-writing the same code each time\n" +
                "2. A `scripts/rotate_pdf.py` script would be helpful to store in the skill\n" +
                "\n" +
                "Example: When designing a `frontend-webapp-builder` skill for queries like \"Build me a todo app\" or \"Build me a dashboard to track my steps,\" the analysis shows:\n" +
                "\n" +
                "1. Writing a frontend webapp requires the same boilerplate HTML/React each time\n" +
                "2. An `assets/hello-world/` template containing the boilerplate HTML/React project files would be helpful to store in the skill\n" +
                "\n" +
                "Example: When building a `big-query` skill to handle queries like \"How many users have logged in today?\" the analysis shows:\n" +
                "\n" +
                "1. Querying BigQuery requires re-discovering the table schemas and relationships each time\n" +
                "2. A `references/schema.md` file documenting the table schemas would be helpful to store in the skill\n" +
                "\n" +
                "To establish the skill's contents, analyze each concrete example to create a list of the reusable resources to include: scripts, references, and assets.\n" +
                "\n" +
                "### Step 3: Initializing the Skill\n" +
                "\n" +
                "At this point, it is time to actually create the skill.\n" +
                "\n" +
                "Skip this step only if the skill being developed already exists, and iteration or packaging is needed. In this case, continue to the next step.\n" +
                "\n" +
                "When creating a new skill from scratch, always run the `init_skill.py` script. The script conveniently generates a new template skill directory that automatically includes everything a skill requires, making the skill creation process much more efficient and reliable.\n" +
                "\n" +
                "Usage:\n" +
                "\n" +
                "```bash\n" +
                "scripts/init_skill.py <skill-name> --path <output-directory>\n" +
                "```\n" +
                "\n" +
                "The script:\n" +
                "\n" +
                "- Creates the skill directory at the specified path\n" +
                "- Generates a SKILL.md template with proper frontmatter and TODO placeholders\n" +
                "- Creates example resource directories: `scripts/`, `references/`, and `assets/`\n" +
                "- Adds example files in each directory that can be customized or deleted\n" +
                "\n" +
                "After initialization, customize or remove the generated SKILL.md and example files as needed.\n" +
                "\n" +
                "### Step 4: Edit the Skill\n" +
                "\n" +
                "When editing the (newly-generated or existing) skill, remember that the skill is being created for another instance of Claude to use. Focus on including information that would be beneficial and non-obvious to Claude. Consider what procedural knowledge, domain-specific details, or reusable assets would help another Claude instance execute these tasks more effectively.\n" +
                "\n" +
                "#### Start with Reusable Skill Contents\n" +
                "\n" +
                "To begin implementation, start with the reusable resources identified above: `scripts/`, `references/`, and `assets/` files. Note that this step may require user input. For example, when implementing a `brand-guidelines` skill, the user may need to provide brand assets or templates to store in `assets/`, or documentation to store in `references/`.\n" +
                "\n" +
                "Also, delete any example files and directories not needed for the skill. The initialization script creates example files in `scripts/`, `references/`, and `assets/` to demonstrate structure, but most skills won't need all of them.\n" +
                "\n" +
                "#### Update SKILL.md\n" +
                "\n" +
                "**Writing Style:** Write the entire skill using **imperative/infinitive form** (verb-first instructions), not second person. Use objective, instructional language (e.g., \"To accomplish X, do Y\" rather than \"You should do X\" or \"If you need to do X\"). This maintains consistency and clarity for AI consumption.\n" +
                "\n" +
                "To complete SKILL.md, answer the following questions:\n" +
                "\n" +
                "1. What is the purpose of the skill, in a few sentences?\n" +
                "2. When should the skill be used?\n" +
                "3. In practice, how should Claude use the skill? All reusable skill contents developed above should be referenced so that Claude knows how to use them.\n" +
                "\n" +
                "### Step 5: Packaging a Skill\n" +
                "\n" +
                "Once the skill is ready, it should be packaged into a distributable zip file that gets shared with the user. The packaging process automatically validates the skill first to ensure it meets all requirements:\n" +
                "\n" +
                "```bash\n" +
                "scripts/package_skill.py <path/to/skill-folder>\n" +
                "```\n" +
                "\n" +
                "Optional output directory specification:\n" +
                "\n" +
                "```bash\n" +
                "scripts/package_skill.py <path/to/skill-folder> ./dist\n" +
                "```\n" +
                "\n" +
                "The packaging script will:\n" +
                "\n" +
                "1. **Validate** the skill automatically, checking:\n" +
                "   - YAML frontmatter format and required fields\n" +
                "   - Skill naming conventions and directory structure\n" +
                "   - Description completeness and quality\n" +
                "   - File organization and resource references\n" +
                "\n" +
                "2. **Package** the skill if validation passes, creating a zip file named after the skill (e.g., `my-skill.zip`) that includes all files and maintains the proper directory structure for distribution.\n" +
                "\n" +
                "If validation fails, the script will report the errors and exit without creating a package. Fix any validation errors and run the packaging command again.\n" +
                "\n" +
                "### Step 6: Iterate\n" +
                "\n" +
                "After testing the skill, users may request improvements. Often this happens right after using the skill, with fresh context of how the skill performed.\n" +
                "\n" +
                "**Iteration workflow:**\n" +
                "1. Use the skill on real tasks\n" +
                "2. Notice struggles or inefficiencies\n" +
                "3. Identify how SKILL.md or bundled resources should be updated\n" +
                "4. Implement changes and test again\n" +
                "The above is the full content of resource `SKILL.md` in skill `skill-creator`\n", content);
    }

    @Test
    public void testFetchSkill1_1() throws Exception {
        FileSystemFetcher fileSystemFetcher = new FileSystemFetcher();
        fileSystemFetcher.setDir(System.getProperty("user.dir") + "/src/test/resources/skills");
        fileSystemFetcher.setPlaceholderResolver(ObjectBuilder.buildEmptyPlaceholderResolver());
        fileSystemFetcher.setResourceService(ObjectBuilder.buildResourceService());
        fileSystemFetcher.setExpire(1000);
        fileSystemFetcher.setUsage("USAGE");
        fileSystemFetcher.setCached(false);
        fileSystemFetcher.init();
        fileSystemFetcher.fetchSkills(ObjectBuilder.buildWorkflowTask());
        String content = fileSystemFetcher.fetchResource(ObjectBuilder.buildWorkflowTask(), "skill-creator", "SKILL.md");
        assertEquals("\n" +
                "The following is the beginning of resource `SKILL.md` in skill `skill-creator`\n" +
                "# Skill Creator\n" +
                "\n" +
                "This skill provides guidance for creating effective skills.\n" +
                "\n" +
                "## About Skills\n" +
                "\n" +
                "Skills are modular, self-contained packages that extend Claude's capabilities by providing\n" +
                "specialized knowledge, workflows, and tools. Think of them as \"onboarding guides\" for specific\n" +
                "domains or tasks—they transform Claude from a general-purpose agent into a specialized agent\n" +
                "equipped with procedural knowledge that no model can fully possess.\n" +
                "\n" +
                "### What Skills Provide\n" +
                "\n" +
                "1. Specialized workflows - Multi-step procedures for specific domains\n" +
                "2. Tool integrations - Instructions for working with specific file formats or APIs\n" +
                "3. Domain expertise - Company-specific knowledge, schemas, business logic\n" +
                "4. Bundled resources - Scripts, references, and assets for complex and repetitive tasks\n" +
                "\n" +
                "### Anatomy of a Skill\n" +
                "\n" +
                "Every skill consists of a required SKILL.md file and optional bundled resources:\n" +
                "\n" +
                "```\n" +
                "skill-name/\n" +
                "├── SKILL.md (required)\n" +
                "│   ├── YAML frontmatter metadata (required)\n" +
                "│   │   ├── name: (required)\n" +
                "│   │   └── description: (required)\n" +
                "│   └── Markdown instructions (required)\n" +
                "└── Bundled Resources (optional)\n" +
                "    ├── scripts/          - Executable code (Python/Bash/etc.)\n" +
                "    ├── references/       - Documentation intended to be loaded into context as needed\n" +
                "    └── assets/           - Files used in output (templates, icons, fonts, etc.)\n" +
                "```\n" +
                "\n" +
                "#### SKILL.md (required)\n" +
                "\n" +
                "**Metadata Quality:** The `name` and `description` in YAML frontmatter determine when Claude will use the skill. Be specific about what the skill does and when to use it. Use the third-person (e.g. \"This skill should be used when...\" instead of \"Use this skill when...\").\n" +
                "\n" +
                "#### Bundled Resources (optional)\n" +
                "\n" +
                "##### Scripts (`scripts/`)\n" +
                "\n" +
                "Executable code (Python/Bash/etc.) for tasks that require deterministic reliability or are repeatedly rewritten.\n" +
                "\n" +
                "- **When to include**: When the same code is being rewritten repeatedly or deterministic reliability is needed\n" +
                "- **Example**: `scripts/rotate_pdf.py` for PDF rotation tasks\n" +
                "- **Benefits**: Token efficient, deterministic, may be executed without loading into context\n" +
                "- **Note**: Scripts may still need to be read by Claude for patching or environment-specific adjustments\n" +
                "\n" +
                "##### References (`references/`)\n" +
                "\n" +
                "Documentation and reference material intended to be loaded as needed into context to inform Claude's process and thinking.\n" +
                "\n" +
                "- **When to include**: For documentation that Claude should reference while working\n" +
                "- **Examples**: `references/finance.md` for financial schemas, `references/mnda.md` for company NDA template, `references/policies.md` for company policies, `references/api_docs.md` for API specifications\n" +
                "- **Use cases**: Database schemas, API documentation, domain knowledge, company policies, detailed workflow guides\n" +
                "- **Benefits**: Keeps SKILL.md lean, loaded only when Claude determines it's needed\n" +
                "- **Best practice**: If files are large (>10k words), include grep search patterns in SKILL.md\n" +
                "- **Avoid duplication**: Information should live in either SKILL.md or references files, not both. Prefer references files for detailed information unless it's truly core to the skill—this keeps SKILL.md lean while making information discoverable without hogging the context window. Keep only essential procedural instructions and workflow guidance in SKILL.md; move detailed reference material, schemas, and examples to references files.\n" +
                "\n" +
                "##### Assets (`assets/`)\n" +
                "\n" +
                "Files not intended to be loaded into context, but rather used within the output Claude produces.\n" +
                "\n" +
                "- **When to include**: When the skill needs files that will be used in the final output\n" +
                "- **Examples**: `assets/logo.png` for brand assets, `assets/slides.pptx` for PowerPoint templates, `assets/frontend-template/` for HTML/React boilerplate, `assets/font.ttf` for typography\n" +
                "- **Use cases**: Templates, images, icons, boilerplate code, fonts, sample documents that get copied or modified\n" +
                "- **Benefits**: Separates output resources from documentation, enables Claude to use files without loading them into context\n" +
                "\n" +
                "### Progressive Disclosure Design Principle\n" +
                "\n" +
                "Skills use a three-level loading system to manage context efficiently:\n" +
                "\n" +
                "1. **Metadata (name + description)** - Always in context (~100 words)\n" +
                "2. **SKILL.md body** - When skill triggers (<5k words)\n" +
                "3. **Bundled resources** - As needed by Claude (Unlimited*)\n" +
                "\n" +
                "*Unlimited because scripts can be executed without reading into context window.\n" +
                "\n" +
                "## Skill Creation Process\n" +
                "\n" +
                "To create a skill, follow the \"Skill Creation Process\" in order, skipping steps only if there is a clear reason why they are not applicable.\n" +
                "\n" +
                "### Step 1: Understanding the Skill with Concrete Examples\n" +
                "\n" +
                "Skip this step only when the skill's usage patterns are already clearly understood. It remains valuable even when working with an existing skill.\n" +
                "\n" +
                "To create an effective skill, clearly understand concrete examples of how the skill will be used. This understanding can come from either direct user examples or generated examples that are validated with user feedback.\n" +
                "\n" +
                "For example, when building an image-editor skill, relevant questions include:\n" +
                "\n" +
                "- \"What functionality should the image-editor skill support? Editing, rotating, anything else?\"\n" +
                "- \"Can you give some examples of how this skill would be used?\"\n" +
                "- \"I can imagine users asking for things like 'Remove the red-eye from this image' or 'Rotate this image'. Are there other ways you imagine this skill being used?\"\n" +
                "- \"What would a user say that should trigger this skill?\"\n" +
                "\n" +
                "To avoid overwhelming users, avoid asking too many questions in a single message. Start with the most important questions and follow up as needed for better effectiveness.\n" +
                "\n" +
                "Conclude this step when there is a clear sense of the functionality the skill should support.\n" +
                "\n" +
                "### Step 2: Planning the Reusable Skill Contents\n" +
                "\n" +
                "To turn concrete examples into an effective skill, analyze each example by:\n" +
                "\n" +
                "1. Considering how to execute on the example from scratch\n" +
                "2. Identifying what scripts, references, and assets would be helpful when executing these workflows repeatedly\n" +
                "\n" +
                "Example: When building a `pdf-editor` skill to handle queries like \"Help me rotate this PDF,\" the analysis shows:\n" +
                "\n" +
                "1. Rotating a PDF requires re-writing the same code each time\n" +
                "2. A `scripts/rotate_pdf.py` script would be helpful to store in the skill\n" +
                "\n" +
                "Example: When designing a `frontend-webapp-builder` skill for queries like \"Build me a todo app\" or \"Build me a dashboard to track my steps,\" the analysis shows:\n" +
                "\n" +
                "1. Writing a frontend webapp requires the same boilerplate HTML/React each time\n" +
                "2. An `assets/hello-world/` template containing the boilerplate HTML/React project files would be helpful to store in the skill\n" +
                "\n" +
                "Example: When building a `big-query` skill to handle queries like \"How many users have logged in today?\" the analysis shows:\n" +
                "\n" +
                "1. Querying BigQuery requires re-discovering the table schemas and relationships each time\n" +
                "2. A `references/schema.md` file documenting the table schemas would be helpful to store in the skill\n" +
                "\n" +
                "To establish the skill's contents, analyze each concrete example to create a list of the reusable resources to include: scripts, references, and assets.\n" +
                "\n" +
                "### Step 3: Initializing the Skill\n" +
                "\n" +
                "At this point, it is time to actually create the skill.\n" +
                "\n" +
                "Skip this step only if the skill being developed already exists, and iteration or packaging is needed. In this case, continue to the next step.\n" +
                "\n" +
                "When creating a new skill from scratch, always run the `init_skill.py` script. The script conveniently generates a new template skill directory that automatically includes everything a skill requires, making the skill creation process much more efficient and reliable.\n" +
                "\n" +
                "Usage:\n" +
                "\n" +
                "```bash\n" +
                "scripts/init_skill.py <skill-name> --path <output-directory>\n" +
                "```\n" +
                "\n" +
                "The script:\n" +
                "\n" +
                "- Creates the skill directory at the specified path\n" +
                "- Generates a SKILL.md template with proper frontmatter and TODO placeholders\n" +
                "- Creates example resource directories: `scripts/`, `references/`, and `assets/`\n" +
                "- Adds example files in each directory that can be customized or deleted\n" +
                "\n" +
                "After initialization, customize or remove the generated SKILL.md and example files as needed.\n" +
                "\n" +
                "### Step 4: Edit the Skill\n" +
                "\n" +
                "When editing the (newly-generated or existing) skill, remember that the skill is being created for another instance of Claude to use. Focus on including information that would be beneficial and non-obvious to Claude. Consider what procedural knowledge, domain-specific details, or reusable assets would help another Claude instance execute these tasks more effectively.\n" +
                "\n" +
                "#### Start with Reusable Skill Contents\n" +
                "\n" +
                "To begin implementation, start with the reusable resources identified above: `scripts/`, `references/`, and `assets/` files. Note that this step may require user input. For example, when implementing a `brand-guidelines` skill, the user may need to provide brand assets or templates to store in `assets/`, or documentation to store in `references/`.\n" +
                "\n" +
                "Also, delete any example files and directories not needed for the skill. The initialization script creates example files in `scripts/`, `references/`, and `assets/` to demonstrate structure, but most skills won't need all of them.\n" +
                "\n" +
                "#### Update SKILL.md\n" +
                "\n" +
                "**Writing Style:** Write the entire skill using **imperative/infinitive form** (verb-first instructions), not second person. Use objective, instructional language (e.g., \"To accomplish X, do Y\" rather than \"You should do X\" or \"If you need to do X\"). This maintains consistency and clarity for AI consumption.\n" +
                "\n" +
                "To complete SKILL.md, answer the following questions:\n" +
                "\n" +
                "1. What is the purpose of the skill, in a few sentences?\n" +
                "2. When should the skill be used?\n" +
                "3. In practice, how should Claude use the skill? All reusable skill contents developed above should be referenced so that Claude knows how to use them.\n" +
                "\n" +
                "### Step 5: Packaging a Skill\n" +
                "\n" +
                "Once the skill is ready, it should be packaged into a distributable zip file that gets shared with the user. The packaging process automatically validates the skill first to ensure it meets all requirements:\n" +
                "\n" +
                "```bash\n" +
                "scripts/package_skill.py <path/to/skill-folder>\n" +
                "```\n" +
                "\n" +
                "Optional output directory specification:\n" +
                "\n" +
                "```bash\n" +
                "scripts/package_skill.py <path/to/skill-folder> ./dist\n" +
                "```\n" +
                "\n" +
                "The packaging script will:\n" +
                "\n" +
                "1. **Validate** the skill automatically, checking:\n" +
                "   - YAML frontmatter format and required fields\n" +
                "   - Skill naming conventions and directory structure\n" +
                "   - Description completeness and quality\n" +
                "   - File organization and resource references\n" +
                "\n" +
                "2. **Package** the skill if validation passes, creating a zip file named after the skill (e.g., `my-skill.zip`) that includes all files and maintains the proper directory structure for distribution.\n" +
                "\n" +
                "If validation fails, the script will report the errors and exit without creating a package. Fix any validation errors and run the packaging command again.\n" +
                "\n" +
                "### Step 6: Iterate\n" +
                "\n" +
                "After testing the skill, users may request improvements. Often this happens right after using the skill, with fresh context of how the skill performed.\n" +
                "\n" +
                "**Iteration workflow:**\n" +
                "1. Use the skill on real tasks\n" +
                "2. Notice struggles or inefficiencies\n" +
                "3. Identify how SKILL.md or bundled resources should be updated\n" +
                "4. Implement changes and test again\n" +
                "The above is the full content of resource `SKILL.md` in skill `skill-creator`\n", content);
    }

    @Test
    public void testFetchSkill2() throws Exception {
        FileSystemFetcher fileSystemFetcher = new FileSystemFetcher();
        fileSystemFetcher.setDir(System.getProperty("user.dir") + "/src/test/resources/skills");
        fileSystemFetcher.setPlaceholderResolver(ObjectBuilder.buildEmptyPlaceholderResolver());
        fileSystemFetcher.setResourceService(ObjectBuilder.buildResourceService());
        fileSystemFetcher.setUsage("USAGE");
        fileSystemFetcher.setExpire(1000);
        fileSystemFetcher.setCached(true);
        fileSystemFetcher.init();
        fileSystemFetcher.fetchSkills(ObjectBuilder.buildWorkflowTask());
        String content = fileSystemFetcher.fetchResource(ObjectBuilder.buildWorkflowTask(), "empty", "SKILL.md");
        assertEquals("\n" +
                "The following is the beginning of resource `SKILL.md` in skill `empty`\n" +
                "# Skill Creator\n" +
                "\n" +
                "This skill provides guidance for creating effective skills.\n" +
                "\n" +
                "## About Skills\n" +
                "\n" +
                "Skills are modular, self-contained packages that extend Claude's capabilities by providing\n" +
                "specialized knowledge, workflows, and tools. Think of them as \"onboarding guides\" for specific\n" +
                "domains or tasks—they transform Claude from a general-purpose agent into a specialized agent\n" +
                "equipped with procedural knowledge that no model can fully possess.\n" +
                "\n" +
                "### What Skills Provide\n" +
                "\n" +
                "1. Specialized workflows - Multi-step procedures for specific domains\n" +
                "2. Tool integrations - Instructions for working with specific file formats or APIs\n" +
                "3. Domain expertise - Company-specific knowledge, schemas, business logic\n" +
                "4. Bundled resources - Scripts, references, and assets for complex and repetitive tasks\n" +
                "\n" +
                "### Anatomy of a Skill\n" +
                "\n" +
                "Every skill consists of a required SKILL.md file and optional bundled resources:\n" +
                "\n" +
                "```\n" +
                "skill-name/\n" +
                "├── SKILL.md (required)\n" +
                "│   ├── YAML frontmatter metadata (required)\n" +
                "│   │   ├── name: (required)\n" +
                "│   │   └── description: (required)\n" +
                "│   └── Markdown instructions (required)\n" +
                "└── Bundled Resources (optional)\n" +
                "    ├── scripts/          - Executable code (Python/Bash/etc.)\n" +
                "    ├── references/       - Documentation intended to be loaded into context as needed\n" +
                "    └── assets/           - Files used in output (templates, icons, fonts, etc.)\n" +
                "```\n" +
                "\n" +
                "#### SKILL.md (required)\n" +
                "\n" +
                "**Metadata Quality:** The `name` and `description` in YAML frontmatter determine when Claude will use the skill. Be specific about what the skill does and when to use it. Use the third-person (e.g. \"This skill should be used when...\" instead of \"Use this skill when...\").\n" +
                "\n" +
                "#### Bundled Resources (optional)\n" +
                "\n" +
                "##### Scripts (`scripts/`)\n" +
                "\n" +
                "Executable code (Python/Bash/etc.) for tasks that require deterministic reliability or are repeatedly rewritten.\n" +
                "\n" +
                "- **When to include**: When the same code is being rewritten repeatedly or deterministic reliability is needed\n" +
                "- **Example**: `scripts/rotate_pdf.py` for PDF rotation tasks\n" +
                "- **Benefits**: Token efficient, deterministic, may be executed without loading into context\n" +
                "- **Note**: Scripts may still need to be read by Claude for patching or environment-specific adjustments\n" +
                "\n" +
                "##### References (`references/`)\n" +
                "\n" +
                "Documentation and reference material intended to be loaded as needed into context to inform Claude's process and thinking.\n" +
                "\n" +
                "- **When to include**: For documentation that Claude should reference while working\n" +
                "- **Examples**: `references/finance.md` for financial schemas, `references/mnda.md` for company NDA template, `references/policies.md` for company policies, `references/api_docs.md` for API specifications\n" +
                "- **Use cases**: Database schemas, API documentation, domain knowledge, company policies, detailed workflow guides\n" +
                "- **Benefits**: Keeps SKILL.md lean, loaded only when Claude determines it's needed\n" +
                "- **Best practice**: If files are large (>10k words), include grep search patterns in SKILL.md\n" +
                "- **Avoid duplication**: Information should live in either SKILL.md or references files, not both. Prefer references files for detailed information unless it's truly core to the skill—this keeps SKILL.md lean while making information discoverable without hogging the context window. Keep only essential procedural instructions and workflow guidance in SKILL.md; move detailed reference material, schemas, and examples to references files.\n" +
                "\n" +
                "##### Assets (`assets/`)\n" +
                "\n" +
                "Files not intended to be loaded into context, but rather used within the output Claude produces.\n" +
                "\n" +
                "- **When to include**: When the skill needs files that will be used in the final output\n" +
                "- **Examples**: `assets/logo.png` for brand assets, `assets/slides.pptx` for PowerPoint templates, `assets/frontend-template/` for HTML/React boilerplate, `assets/font.ttf` for typography\n" +
                "- **Use cases**: Templates, images, icons, boilerplate code, fonts, sample documents that get copied or modified\n" +
                "- **Benefits**: Separates output resources from documentation, enables Claude to use files without loading them into context\n" +
                "\n" +
                "### Progressive Disclosure Design Principle\n" +
                "\n" +
                "Skills use a three-level loading system to manage context efficiently:\n" +
                "\n" +
                "1. **Metadata (name + description)** - Always in context (~100 words)\n" +
                "2. **SKILL.md body** - When skill triggers (<5k words)\n" +
                "3. **Bundled resources** - As needed by Claude (Unlimited*)\n" +
                "\n" +
                "*Unlimited because scripts can be executed without reading into context window.\n" +
                "\n" +
                "## Skill Creation Process\n" +
                "\n" +
                "To create a skill, follow the \"Skill Creation Process\" in order, skipping steps only if there is a clear reason why they are not applicable.\n" +
                "\n" +
                "### Step 1: Understanding the Skill with Concrete Examples\n" +
                "\n" +
                "Skip this step only when the skill's usage patterns are already clearly understood. It remains valuable even when working with an existing skill.\n" +
                "\n" +
                "To create an effective skill, clearly understand concrete examples of how the skill will be used. This understanding can come from either direct user examples or generated examples that are validated with user feedback.\n" +
                "\n" +
                "For example, when building an image-editor skill, relevant questions include:\n" +
                "\n" +
                "- \"What functionality should the image-editor skill support? Editing, rotating, anything else?\"\n" +
                "- \"Can you give some examples of how this skill would be used?\"\n" +
                "- \"I can imagine users asking for things like 'Remove the red-eye from this image' or 'Rotate this image'. Are there other ways you imagine this skill being used?\"\n" +
                "- \"What would a user say that should trigger this skill?\"\n" +
                "\n" +
                "To avoid overwhelming users, avoid asking too many questions in a single message. Start with the most important questions and follow up as needed for better effectiveness.\n" +
                "\n" +
                "Conclude this step when there is a clear sense of the functionality the skill should support.\n" +
                "\n" +
                "### Step 2: Planning the Reusable Skill Contents\n" +
                "\n" +
                "To turn concrete examples into an effective skill, analyze each example by:\n" +
                "\n" +
                "1. Considering how to execute on the example from scratch\n" +
                "2. Identifying what scripts, references, and assets would be helpful when executing these workflows repeatedly\n" +
                "\n" +
                "Example: When building a `pdf-editor` skill to handle queries like \"Help me rotate this PDF,\" the analysis shows:\n" +
                "\n" +
                "1. Rotating a PDF requires re-writing the same code each time\n" +
                "2. A `scripts/rotate_pdf.py` script would be helpful to store in the skill\n" +
                "\n" +
                "Example: When designing a `frontend-webapp-builder` skill for queries like \"Build me a todo app\" or \"Build me a dashboard to track my steps,\" the analysis shows:\n" +
                "\n" +
                "1. Writing a frontend webapp requires the same boilerplate HTML/React each time\n" +
                "2. An `assets/hello-world/` template containing the boilerplate HTML/React project files would be helpful to store in the skill\n" +
                "\n" +
                "Example: When building a `big-query` skill to handle queries like \"How many users have logged in today?\" the analysis shows:\n" +
                "\n" +
                "1. Querying BigQuery requires re-discovering the table schemas and relationships each time\n" +
                "2. A `references/schema.md` file documenting the table schemas would be helpful to store in the skill\n" +
                "\n" +
                "To establish the skill's contents, analyze each concrete example to create a list of the reusable resources to include: scripts, references, and assets.\n" +
                "\n" +
                "### Step 3: Initializing the Skill\n" +
                "\n" +
                "At this point, it is time to actually create the skill.\n" +
                "\n" +
                "Skip this step only if the skill being developed already exists, and iteration or packaging is needed. In this case, continue to the next step.\n" +
                "\n" +
                "When creating a new skill from scratch, always run the `init_skill.py` script. The script conveniently generates a new template skill directory that automatically includes everything a skill requires, making the skill creation process much more efficient and reliable.\n" +
                "\n" +
                "Usage:\n" +
                "\n" +
                "```bash\n" +
                "scripts/init_skill.py <skill-name> --path <output-directory>\n" +
                "```\n" +
                "\n" +
                "The script:\n" +
                "\n" +
                "- Creates the skill directory at the specified path\n" +
                "- Generates a SKILL.md template with proper frontmatter and TODO placeholders\n" +
                "- Creates example resource directories: `scripts/`, `references/`, and `assets/`\n" +
                "- Adds example files in each directory that can be customized or deleted\n" +
                "\n" +
                "After initialization, customize or remove the generated SKILL.md and example files as needed.\n" +
                "\n" +
                "### Step 4: Edit the Skill\n" +
                "\n" +
                "When editing the (newly-generated or existing) skill, remember that the skill is being created for another instance of Claude to use. Focus on including information that would be beneficial and non-obvious to Claude. Consider what procedural knowledge, domain-specific details, or reusable assets would help another Claude instance execute these tasks more effectively.\n" +
                "\n" +
                "#### Start with Reusable Skill Contents\n" +
                "\n" +
                "To begin implementation, start with the reusable resources identified above: `scripts/`, `references/`, and `assets/` files. Note that this step may require user input. For example, when implementing a `brand-guidelines` skill, the user may need to provide brand assets or templates to store in `assets/`, or documentation to store in `references/`.\n" +
                "\n" +
                "Also, delete any example files and directories not needed for the skill. The initialization script creates example files in `scripts/`, `references/`, and `assets/` to demonstrate structure, but most skills won't need all of them.\n" +
                "\n" +
                "#### Update SKILL.md\n" +
                "\n" +
                "**Writing Style:** Write the entire skill using **imperative/infinitive form** (verb-first instructions), not second person. Use objective, instructional language (e.g., \"To accomplish X, do Y\" rather than \"You should do X\" or \"If you need to do X\"). This maintains consistency and clarity for AI consumption.\n" +
                "\n" +
                "To complete SKILL.md, answer the following questions:\n" +
                "\n" +
                "1. What is the purpose of the skill, in a few sentences?\n" +
                "2. When should the skill be used?\n" +
                "3. In practice, how should Claude use the skill? All reusable skill contents developed above should be referenced so that Claude knows how to use them.\n" +
                "\n" +
                "### Step 5: Packaging a Skill\n" +
                "\n" +
                "Once the skill is ready, it should be packaged into a distributable zip file that gets shared with the user. The packaging process automatically validates the skill first to ensure it meets all requirements:\n" +
                "\n" +
                "```bash\n" +
                "scripts/package_skill.py <path/to/skill-folder>\n" +
                "```\n" +
                "\n" +
                "Optional output directory specification:\n" +
                "\n" +
                "```bash\n" +
                "scripts/package_skill.py <path/to/skill-folder> ./dist\n" +
                "```\n" +
                "\n" +
                "The packaging script will:\n" +
                "\n" +
                "1. **Validate** the skill automatically, checking:\n" +
                "   - YAML frontmatter format and required fields\n" +
                "   - Skill naming conventions and directory structure\n" +
                "   - Description completeness and quality\n" +
                "   - File organization and resource references\n" +
                "\n" +
                "2. **Package** the skill if validation passes, creating a zip file named after the skill (e.g., `my-skill.zip`) that includes all files and maintains the proper directory structure for distribution.\n" +
                "\n" +
                "If validation fails, the script will report the errors and exit without creating a package. Fix any validation errors and run the packaging command again.\n" +
                "\n" +
                "### Step 6: Iterate\n" +
                "\n" +
                "After testing the skill, users may request improvements. Often this happens right after using the skill, with fresh context of how the skill performed.\n" +
                "\n" +
                "**Iteration workflow:**\n" +
                "1. Use the skill on real tasks\n" +
                "2. Notice struggles or inefficiencies\n" +
                "3. Identify how SKILL.md or bundled resources should be updated\n" +
                "4. Implement changes and test again\n" +
                "The above is the full content of resource `SKILL.md` in skill `empty`\n", content);
    }

    @Test
    public void testFetchSkill3() throws Exception {
        FileSystemFetcher fileSystemFetcher = new FileSystemFetcher();
        fileSystemFetcher.setDir(System.getProperty("user.dir") + "/src/test/resources/skills");
        fileSystemFetcher.setPlaceholderResolver(ObjectBuilder.buildEmptyPlaceholderResolver());
        fileSystemFetcher.setResourceService(ObjectBuilder.buildResourceService());
        fileSystemFetcher.setUsage("USAGE");
        fileSystemFetcher.setExpire(1000);
        fileSystemFetcher.setCached(true);
        fileSystemFetcher.init();
        fileSystemFetcher.fetchSkills(ObjectBuilder.buildWorkflowTask());
        String content = fileSystemFetcher.fetchResource(ObjectBuilder.buildWorkflowTask(), "skill-creator", "SKILL.md");
        assertEquals("\n" +
                "The following is the beginning of resource `SKILL.md` in skill `skill-creator`\n" +
                "# Skill Creator\n" +
                "\n" +
                "This skill provides guidance for creating effective skills.\n" +
                "\n" +
                "## About Skills\n" +
                "\n" +
                "Skills are modular, self-contained packages that extend Claude's capabilities by providing\n" +
                "specialized knowledge, workflows, and tools. Think of them as \"onboarding guides\" for specific\n" +
                "domains or tasks—they transform Claude from a general-purpose agent into a specialized agent\n" +
                "equipped with procedural knowledge that no model can fully possess.\n" +
                "\n" +
                "### What Skills Provide\n" +
                "\n" +
                "1. Specialized workflows - Multi-step procedures for specific domains\n" +
                "2. Tool integrations - Instructions for working with specific file formats or APIs\n" +
                "3. Domain expertise - Company-specific knowledge, schemas, business logic\n" +
                "4. Bundled resources - Scripts, references, and assets for complex and repetitive tasks\n" +
                "\n" +
                "### Anatomy of a Skill\n" +
                "\n" +
                "Every skill consists of a required SKILL.md file and optional bundled resources:\n" +
                "\n" +
                "```\n" +
                "skill-name/\n" +
                "├── SKILL.md (required)\n" +
                "│   ├── YAML frontmatter metadata (required)\n" +
                "│   │   ├── name: (required)\n" +
                "│   │   └── description: (required)\n" +
                "│   └── Markdown instructions (required)\n" +
                "└── Bundled Resources (optional)\n" +
                "    ├── scripts/          - Executable code (Python/Bash/etc.)\n" +
                "    ├── references/       - Documentation intended to be loaded into context as needed\n" +
                "    └── assets/           - Files used in output (templates, icons, fonts, etc.)\n" +
                "```\n" +
                "\n" +
                "#### SKILL.md (required)\n" +
                "\n" +
                "**Metadata Quality:** The `name` and `description` in YAML frontmatter determine when Claude will use the skill. Be specific about what the skill does and when to use it. Use the third-person (e.g. \"This skill should be used when...\" instead of \"Use this skill when...\").\n" +
                "\n" +
                "#### Bundled Resources (optional)\n" +
                "\n" +
                "##### Scripts (`scripts/`)\n" +
                "\n" +
                "Executable code (Python/Bash/etc.) for tasks that require deterministic reliability or are repeatedly rewritten.\n" +
                "\n" +
                "- **When to include**: When the same code is being rewritten repeatedly or deterministic reliability is needed\n" +
                "- **Example**: `scripts/rotate_pdf.py` for PDF rotation tasks\n" +
                "- **Benefits**: Token efficient, deterministic, may be executed without loading into context\n" +
                "- **Note**: Scripts may still need to be read by Claude for patching or environment-specific adjustments\n" +
                "\n" +
                "##### References (`references/`)\n" +
                "\n" +
                "Documentation and reference material intended to be loaded as needed into context to inform Claude's process and thinking.\n" +
                "\n" +
                "- **When to include**: For documentation that Claude should reference while working\n" +
                "- **Examples**: `references/finance.md` for financial schemas, `references/mnda.md` for company NDA template, `references/policies.md` for company policies, `references/api_docs.md` for API specifications\n" +
                "- **Use cases**: Database schemas, API documentation, domain knowledge, company policies, detailed workflow guides\n" +
                "- **Benefits**: Keeps SKILL.md lean, loaded only when Claude determines it's needed\n" +
                "- **Best practice**: If files are large (>10k words), include grep search patterns in SKILL.md\n" +
                "- **Avoid duplication**: Information should live in either SKILL.md or references files, not both. Prefer references files for detailed information unless it's truly core to the skill—this keeps SKILL.md lean while making information discoverable without hogging the context window. Keep only essential procedural instructions and workflow guidance in SKILL.md; move detailed reference material, schemas, and examples to references files.\n" +
                "\n" +
                "##### Assets (`assets/`)\n" +
                "\n" +
                "Files not intended to be loaded into context, but rather used within the output Claude produces.\n" +
                "\n" +
                "- **When to include**: When the skill needs files that will be used in the final output\n" +
                "- **Examples**: `assets/logo.png` for brand assets, `assets/slides.pptx` for PowerPoint templates, `assets/frontend-template/` for HTML/React boilerplate, `assets/font.ttf` for typography\n" +
                "- **Use cases**: Templates, images, icons, boilerplate code, fonts, sample documents that get copied or modified\n" +
                "- **Benefits**: Separates output resources from documentation, enables Claude to use files without loading them into context\n" +
                "\n" +
                "### Progressive Disclosure Design Principle\n" +
                "\n" +
                "Skills use a three-level loading system to manage context efficiently:\n" +
                "\n" +
                "1. **Metadata (name + description)** - Always in context (~100 words)\n" +
                "2. **SKILL.md body** - When skill triggers (<5k words)\n" +
                "3. **Bundled resources** - As needed by Claude (Unlimited*)\n" +
                "\n" +
                "*Unlimited because scripts can be executed without reading into context window.\n" +
                "\n" +
                "## Skill Creation Process\n" +
                "\n" +
                "To create a skill, follow the \"Skill Creation Process\" in order, skipping steps only if there is a clear reason why they are not applicable.\n" +
                "\n" +
                "### Step 1: Understanding the Skill with Concrete Examples\n" +
                "\n" +
                "Skip this step only when the skill's usage patterns are already clearly understood. It remains valuable even when working with an existing skill.\n" +
                "\n" +
                "To create an effective skill, clearly understand concrete examples of how the skill will be used. This understanding can come from either direct user examples or generated examples that are validated with user feedback.\n" +
                "\n" +
                "For example, when building an image-editor skill, relevant questions include:\n" +
                "\n" +
                "- \"What functionality should the image-editor skill support? Editing, rotating, anything else?\"\n" +
                "- \"Can you give some examples of how this skill would be used?\"\n" +
                "- \"I can imagine users asking for things like 'Remove the red-eye from this image' or 'Rotate this image'. Are there other ways you imagine this skill being used?\"\n" +
                "- \"What would a user say that should trigger this skill?\"\n" +
                "\n" +
                "To avoid overwhelming users, avoid asking too many questions in a single message. Start with the most important questions and follow up as needed for better effectiveness.\n" +
                "\n" +
                "Conclude this step when there is a clear sense of the functionality the skill should support.\n" +
                "\n" +
                "### Step 2: Planning the Reusable Skill Contents\n" +
                "\n" +
                "To turn concrete examples into an effective skill, analyze each example by:\n" +
                "\n" +
                "1. Considering how to execute on the example from scratch\n" +
                "2. Identifying what scripts, references, and assets would be helpful when executing these workflows repeatedly\n" +
                "\n" +
                "Example: When building a `pdf-editor` skill to handle queries like \"Help me rotate this PDF,\" the analysis shows:\n" +
                "\n" +
                "1. Rotating a PDF requires re-writing the same code each time\n" +
                "2. A `scripts/rotate_pdf.py` script would be helpful to store in the skill\n" +
                "\n" +
                "Example: When designing a `frontend-webapp-builder` skill for queries like \"Build me a todo app\" or \"Build me a dashboard to track my steps,\" the analysis shows:\n" +
                "\n" +
                "1. Writing a frontend webapp requires the same boilerplate HTML/React each time\n" +
                "2. An `assets/hello-world/` template containing the boilerplate HTML/React project files would be helpful to store in the skill\n" +
                "\n" +
                "Example: When building a `big-query` skill to handle queries like \"How many users have logged in today?\" the analysis shows:\n" +
                "\n" +
                "1. Querying BigQuery requires re-discovering the table schemas and relationships each time\n" +
                "2. A `references/schema.md` file documenting the table schemas would be helpful to store in the skill\n" +
                "\n" +
                "To establish the skill's contents, analyze each concrete example to create a list of the reusable resources to include: scripts, references, and assets.\n" +
                "\n" +
                "### Step 3: Initializing the Skill\n" +
                "\n" +
                "At this point, it is time to actually create the skill.\n" +
                "\n" +
                "Skip this step only if the skill being developed already exists, and iteration or packaging is needed. In this case, continue to the next step.\n" +
                "\n" +
                "When creating a new skill from scratch, always run the `init_skill.py` script. The script conveniently generates a new template skill directory that automatically includes everything a skill requires, making the skill creation process much more efficient and reliable.\n" +
                "\n" +
                "Usage:\n" +
                "\n" +
                "```bash\n" +
                "scripts/init_skill.py <skill-name> --path <output-directory>\n" +
                "```\n" +
                "\n" +
                "The script:\n" +
                "\n" +
                "- Creates the skill directory at the specified path\n" +
                "- Generates a SKILL.md template with proper frontmatter and TODO placeholders\n" +
                "- Creates example resource directories: `scripts/`, `references/`, and `assets/`\n" +
                "- Adds example files in each directory that can be customized or deleted\n" +
                "\n" +
                "After initialization, customize or remove the generated SKILL.md and example files as needed.\n" +
                "\n" +
                "### Step 4: Edit the Skill\n" +
                "\n" +
                "When editing the (newly-generated or existing) skill, remember that the skill is being created for another instance of Claude to use. Focus on including information that would be beneficial and non-obvious to Claude. Consider what procedural knowledge, domain-specific details, or reusable assets would help another Claude instance execute these tasks more effectively.\n" +
                "\n" +
                "#### Start with Reusable Skill Contents\n" +
                "\n" +
                "To begin implementation, start with the reusable resources identified above: `scripts/`, `references/`, and `assets/` files. Note that this step may require user input. For example, when implementing a `brand-guidelines` skill, the user may need to provide brand assets or templates to store in `assets/`, or documentation to store in `references/`.\n" +
                "\n" +
                "Also, delete any example files and directories not needed for the skill. The initialization script creates example files in `scripts/`, `references/`, and `assets/` to demonstrate structure, but most skills won't need all of them.\n" +
                "\n" +
                "#### Update SKILL.md\n" +
                "\n" +
                "**Writing Style:** Write the entire skill using **imperative/infinitive form** (verb-first instructions), not second person. Use objective, instructional language (e.g., \"To accomplish X, do Y\" rather than \"You should do X\" or \"If you need to do X\"). This maintains consistency and clarity for AI consumption.\n" +
                "\n" +
                "To complete SKILL.md, answer the following questions:\n" +
                "\n" +
                "1. What is the purpose of the skill, in a few sentences?\n" +
                "2. When should the skill be used?\n" +
                "3. In practice, how should Claude use the skill? All reusable skill contents developed above should be referenced so that Claude knows how to use them.\n" +
                "\n" +
                "### Step 5: Packaging a Skill\n" +
                "\n" +
                "Once the skill is ready, it should be packaged into a distributable zip file that gets shared with the user. The packaging process automatically validates the skill first to ensure it meets all requirements:\n" +
                "\n" +
                "```bash\n" +
                "scripts/package_skill.py <path/to/skill-folder>\n" +
                "```\n" +
                "\n" +
                "Optional output directory specification:\n" +
                "\n" +
                "```bash\n" +
                "scripts/package_skill.py <path/to/skill-folder> ./dist\n" +
                "```\n" +
                "\n" +
                "The packaging script will:\n" +
                "\n" +
                "1. **Validate** the skill automatically, checking:\n" +
                "   - YAML frontmatter format and required fields\n" +
                "   - Skill naming conventions and directory structure\n" +
                "   - Description completeness and quality\n" +
                "   - File organization and resource references\n" +
                "\n" +
                "2. **Package** the skill if validation passes, creating a zip file named after the skill (e.g., `my-skill.zip`) that includes all files and maintains the proper directory structure for distribution.\n" +
                "\n" +
                "If validation fails, the script will report the errors and exit without creating a package. Fix any validation errors and run the packaging command again.\n" +
                "\n" +
                "### Step 6: Iterate\n" +
                "\n" +
                "After testing the skill, users may request improvements. Often this happens right after using the skill, with fresh context of how the skill performed.\n" +
                "\n" +
                "**Iteration workflow:**\n" +
                "1. Use the skill on real tasks\n" +
                "2. Notice struggles or inefficiencies\n" +
                "3. Identify how SKILL.md or bundled resources should be updated\n" +
                "4. Implement changes and test again\n" +
                "The above is the full content of resource `SKILL.md` in skill `skill-creator`\n", content);
    }

    @Test
    public void testFetchSkill4() throws Exception {
        FileSystemFetcher fileSystemFetcher = new FileSystemFetcher();
        fileSystemFetcher.setDir(System.getProperty("user.dir") + "/src/test/resources/skills");
        fileSystemFetcher.setPlaceholderResolver(ObjectBuilder.buildEmptyPlaceholderResolver());
        fileSystemFetcher.setResourceService(ObjectBuilder.buildResourceService());
        fileSystemFetcher.setUsage("USAGE");
        fileSystemFetcher.setExpire(1000);
        fileSystemFetcher.setCached(true);
        fileSystemFetcher.init();
        fileSystemFetcher.fetchSkills(ObjectBuilder.buildWorkflowTask());
        fileSystemFetcher.fetchResource(ObjectBuilder.buildWorkflowTask(), "skill-creator", "SKILL.md");
    }


    @Test
    public void testBuildPath1() throws Exception {
        FileSystemFetcher fileSystemFetcher = new FileSystemFetcher();
        fileSystemFetcher.setDir(System.getProperty("user.dir") + "/src/test/resources/skills");
        fileSystemFetcher.setPlaceholderResolver(ObjectBuilder.buildEmptyPlaceholderResolver());
        fileSystemFetcher.setResourceService(ObjectBuilder.buildResourceService());
        fileSystemFetcher.setExpire(1000);
        fileSystemFetcher.init();
        Assert.assertTrue(fileSystemFetcher.buildPath().endsWith("/src/test/resources/skills"));
    }

    @Test
    public void testBuildPath2() throws Exception {
        FileSystemFetcher fileSystemFetcher = new FileSystemFetcher();
        fileSystemFetcher.setDir(System.getProperty("user.dir") + "/src/test/resources/skills");
        fileSystemFetcher.setPlaceholderResolver(ObjectBuilder.buildEmptyPlaceholderResolver());
        fileSystemFetcher.setResourceService(ObjectBuilder.buildResourceService());
        fileSystemFetcher.setExpire(1000);
        fileSystemFetcher.init();
        Assert.assertTrue(fileSystemFetcher.buildPath().endsWith("/src/test/resources/skills"));
    }

    /**
     * 覆盖 buildDef()：未设置 dir 时走默认路径，返回 classloader 中 resource \"skills\" 的路径。
     */
    @Test
    public void testBuildDef() throws Exception {
        FileSystemFetcher fetcher = new FileSystemFetcher();
        fetcher.setResourceService(ObjectBuilder.buildResourceService());
        fetcher.setRelease(false);
        fetcher.setExpire(1000);
        fetcher.init();
        String path = (String) FileSystemFetcher.class.getDeclaredMethod("buildDef").invoke(fetcher);
        Assert.assertNotNull(path);
        Assert.assertTrue("buildDef() 应返回包含 skills 的路径", path.contains("skills"));
    }

    /**
     * 覆盖 buildDef() 的 release 且 nested 分支：URL 路径以 nested: 开头时，从 resourceReleaser.getRoot() 拼 BOOT-INF/classes/skills。
     */
    @Test
    public void testBuildDefReleaseNested() throws Exception {
        URL nestedUrl = new URL("file", "", "nested:app/skills");
        ClassLoader parent = ObjectBuilder.class.getClassLoader();
        ClassLoader loader = new ClassLoader(parent) {
            @Override
            public URL getResource(String name) {
                if ("skills".equals(name)) return nestedUrl;
                return super.getResource(name);
            }

            @Override
            protected Class<?> loadClass(String name, boolean resolve) throws ClassNotFoundException {
                if (ObjectBuilder.class.getName().equals(name)) {
                    synchronized (getClassLoadingLock(name)) {
                        Class<?> c = findLoadedClass(name);
                        if (c == null) {
                            String path = name.replace('.', '/') + ".class";
                            try (java.io.InputStream in = getParent().getResourceAsStream(path)) {
                                if (in != null) {
                                    byte[] bytes = org.apache.commons.io.IOUtils.toByteArray(in);
                                    c = defineClass(name, bytes, 0, bytes.length);
                                }
                            } catch (Exception e) {
                                throw new ClassNotFoundException(name, e);
                            }
                        }
                        if (resolve) resolveClass(c);
                        return c;
                    }
                }
                return super.loadClass(name, resolve);
            }
        };
        Class<?> rootClass = loader.loadClass(ObjectBuilder.class.getName());
        Assert.assertSame(loader, rootClass.getClassLoader());
        ResourceService resourceService = new ResourceService() {
            @Override
            public URL url(String location) throws Exception {
                return ObjectBuilder.buildResourceService().url(location);
            }

            @Override
            public Class<?> root() {
                return rootClass;
            }
        };
        String jarRoot = "/tmp/app.jar";
        ResourceReleaser resourceReleaser = new ResourceReleaser() {
            @Override
            public String getRoot() {
                return jarRoot;
            }
        };
        FileSystemFetcher fetcher = new FileSystemFetcher();
        fetcher.setResourceService(resourceService);
        fetcher.setResourceReleaser(resourceReleaser);
        fetcher.setRelease(true);
        fetcher.setExpire(1000);
        fetcher.init();
        java.lang.reflect.Method buildDef = FileSystemFetcher.class.getDeclaredMethod("buildDef");
        buildDef.setAccessible(true);
        String path = (String) buildDef.invoke(fetcher);
        Assert.assertNotNull(path);
        Assert.assertTrue("release+nested 时应包含 BOOT-INF", path.contains("BOOT-INF"));
        Assert.assertTrue(path.contains("classes"));
        Assert.assertTrue(path.contains("skills"));
        Assert.assertEquals(
                Paths.get("/tmp/app", "BOOT-INF", "classes", "skills").toString(),
                path);
    }

    /**
     * 覆盖 buildDef() 中 Assert.notNull(url, ...)：当 getResource(\"skills\") 返回 null 时抛出异常。
     */
    @Test(expected = IllegalArgumentException.class)
    public void testBuildDef_throwsWhenResourceNotFound() throws Throwable {
        ClassLoader parent = ObjectBuilder.class.getClassLoader();
        ClassLoader loader = new ClassLoader(parent) {
            @Override
            public URL getResource(String name) {
                return "skills".equals(name) ? null : super.getResource(name);
            }

            @Override
            protected Class<?> loadClass(String name, boolean resolve) throws ClassNotFoundException {
                if (ObjectBuilder.class.getName().equals(name)) {
                    synchronized (getClassLoadingLock(name)) {
                        Class<?> c = findLoadedClass(name);
                        if (c == null) {
                            String path = name.replace('.', '/') + ".class";
                            try (java.io.InputStream in = getParent().getResourceAsStream(path)) {
                                if (in != null) {
                                    byte[] bytes = org.apache.commons.io.IOUtils.toByteArray(in);
                                    c = defineClass(name, bytes, 0, bytes.length);
                                }
                            } catch (Exception e) {
                                throw new ClassNotFoundException(name, e);
                            }
                        }
                        if (resolve) resolveClass(c);
                        return c;
                    }
                }
                return super.loadClass(name, resolve);
            }
        };
        Class<?> rootClass = loader.loadClass(ObjectBuilder.class.getName());
        ResourceService resourceService = new ResourceService() {
            @Override
            public URL url(String location) throws Exception {
                return ObjectBuilder.buildResourceService().url(location);
            }

            @Override
            public Class<?> root() {
                return rootClass;
            }
        };
        FileSystemFetcher fetcher = new FileSystemFetcher();
        fetcher.setResourceService(resourceService);
        fetcher.setRelease(false);
        fetcher.setExpire(1000);
        fetcher.init();
        java.lang.reflect.Method buildDef = FileSystemFetcher.class.getDeclaredMethod("buildDef");
        buildDef.setAccessible(true);
        try {
            buildDef.invoke(fetcher);
        } catch (java.lang.reflect.InvocationTargetException e) {
            throw e.getCause();
        }
    }

    @Test
    public void testFetchMetadataWithException() throws Exception {
        FileSystemFetcher fileSystemFetcher = new FileSystemFetcher() {
            @Override
            protected SkillVisitor buildVisitor() throws Exception {
                throw new RuntimeException();
            }
        };
        fileSystemFetcher.setPlaceholderResolver(ObjectBuilder.buildEmptyPlaceholderResolver());
        fileSystemFetcher.setDir(System.getProperty("user.dir") + "/src/test/resources/skills");
        fileSystemFetcher.setResourceService(ObjectBuilder.buildResourceService());
        fileSystemFetcher.setUsage("USAGE");
        fileSystemFetcher.setExpire(1000);
        fileSystemFetcher.setCached(false);
        fileSystemFetcher.init();
        assertEquals(Integer.valueOf(0), Integer.valueOf(fileSystemFetcher.fetchSkills(ObjectBuilder.buildWorkflowTask()).getSkills().size()));
    }

    @Test
    public void testRemoveHeaderNoHeader() throws Exception {
        FileSystemFetcher fetcher = new FileSystemFetcher();
        fetcher.setPlaceholderResolver(ObjectBuilder.buildEmptyPlaceholderResolver());
        String content = "No Header Content";
        assertEquals(content, fetcher.removeHeader(content));
    }

    @Test
    public void testRemoveHeaderWithYamlBlock() throws Exception {
        FileSystemFetcher fetcher = new FileSystemFetcher();
        fetcher.setPlaceholderResolver(ObjectBuilder.buildEmptyPlaceholderResolver());
        String content = "---\nname: foo\n---\n\n# Body\nMarkdown body here.";
        assertEquals("# Body\nMarkdown body here.", fetcher.removeHeader(content));
    }

    @Test
    public void testRemoveHeaderNoMatch() throws Exception {
        FileSystemFetcher fetcher = new FileSystemFetcher();
        fetcher.setPlaceholderResolver(ObjectBuilder.buildEmptyPlaceholderResolver());
        String content = "plain content without yaml header";
        assertEquals(content, fetcher.removeHeader(content));
    }

    @Test
    public void testRelaceContent1() throws Exception {
        FileSystemFetcher fileSystemFetcher = new FileSystemFetcher();
        fileSystemFetcher.setPlaceholderResolver(ObjectBuilder.buildEmptyPlaceholderResolver());
        assertEquals("HELLO", fileSystemFetcher.replaceContent(ObjectBuilder.buildWorkflowTask(), "", "", "HELLO"));
    }

    @Test
    public void testRelaceContent2() throws Exception {
        FileSystemFetcher fileSystemFetcher = new FileSystemFetcher();
        fileSystemFetcher.setPlaceholderResolver(ObjectBuilder.buildEmptyPlaceholderResolver());
        fileSystemFetcher.setPrefix("#");
        fileSystemFetcher.setArgs("ARGS");
        WorkflowTask workflowTask = ObjectBuilder.buildWorkflowTask();
        workflowTask.putMetadata(fileSystemFetcher.getArgs(), ImmutableMap.of("${L}", "B", "#W", "D"));
        assertEquals("HEBLO DORLD", fileSystemFetcher.replaceContent(workflowTask, "", "", "HE#${L}LO #WORLD"));
    }

    @Test
    public void testRelaceContentWithException() throws Exception {
        FileSystemFetcher fileSystemFetcher = new FileSystemFetcher();
        fileSystemFetcher.setPlaceholderResolver(ObjectBuilder.buildEmptyPlaceholderResolver());
        fileSystemFetcher.setPrefix("#");
        fileSystemFetcher.setArgs("ARGS");
        WorkflowTask workflowTask = ObjectBuilder.buildWorkflowTask();
        workflowTask.putMetadata(fileSystemFetcher.getArgs(), new String[]{});
        assertEquals("HE#${L}LO #WORLD", fileSystemFetcher.replaceContent(workflowTask, "", "", "HE#${L}LO #WORLD"));
    }

    @Test
    public void testRelaceContentWithEmpty() throws Exception {
        FileSystemFetcher fileSystemFetcher = new FileSystemFetcher();
        fileSystemFetcher.setPlaceholderResolver(ObjectBuilder.buildEmptyPlaceholderResolver());
        fileSystemFetcher.setPrefix("#");
        fileSystemFetcher.setArgs("ARGS");
        WorkflowTask workflowTask = ObjectBuilder.buildWorkflowTask();
        workflowTask.putMetadata(fileSystemFetcher.getArgs(), new HashMap<>());
        assertEquals("HE#${L}LO #WORLD", fileSystemFetcher.replaceContent(workflowTask, "", "", "HE#${L}LO #WORLD"));
    }

    @Test
    public void testReplaceContentWithNonStringValues() throws Exception {
        FileSystemFetcher fileSystemFetcher = new FileSystemFetcher();
        fileSystemFetcher.setPlaceholderResolver(ObjectBuilder.buildEmptyPlaceholderResolver());
        fileSystemFetcher.setPrefix("#");
        fileSystemFetcher.setArgs("ARGS");
        WorkflowTask workflowTask = ObjectBuilder.buildWorkflowTask();
        Map<String, Object> args = new LinkedHashMap<>();
        args.put("${NUM}", 123);
        args.put("${MAP}", Collections.singletonMap("a", "b"));
        args.put("${STR}", "plain");
        workflowTask.putMetadata(fileSystemFetcher.getArgs(), args);
        String result = fileSystemFetcher.replaceContent(workflowTask, "", "", "num=#${NUM} map=#${MAP} str=#${STR}");
        assertEquals("num=123 map=" + Collections.singletonMap("a", "b").toString() + " str=plain", result);
    }

    /**
     * args 为空时使用 workTask.getMetadata() 全量替换
     */
    @Test
    public void testReplaceContentWithArgsEmptyAndMetadataNotEmpty() throws Exception {
        FileSystemFetcher fileSystemFetcher = new FileSystemFetcher();
        fileSystemFetcher.setPlaceholderResolver(ObjectBuilder.buildEmptyPlaceholderResolver());
        fileSystemFetcher.setPrefix("#");
        fileSystemFetcher.setArgs("");
        WorkflowTask workflowTask = ObjectBuilder.buildWorkflowTask();
        workflowTask.getMetadata().put("${KEY}", "VAL");
        String result = fileSystemFetcher.replaceContent(workflowTask, "", "", "prefix_#${KEY}_suffix");
        assertEquals("prefix_VAL_suffix", result);
    }

    /**
     * args 为 null 时使用 workTask.getMetadata() 全量替换
     */
    @Test
    public void testReplaceContentWithArgsNullAndMetadataNotEmpty() throws Exception {
        FileSystemFetcher fileSystemFetcher = new FileSystemFetcher();
        fileSystemFetcher.setPlaceholderResolver(ObjectBuilder.buildEmptyPlaceholderResolver());
        fileSystemFetcher.setPrefix("#");
        fileSystemFetcher.setArgs(null);
        WorkflowTask workflowTask = ObjectBuilder.buildWorkflowTask();
        workflowTask.getMetadata().put("${A}", "1");
        workflowTask.getMetadata().put("${B}", "2");
        String result = fileSystemFetcher.replaceContent(workflowTask, "", "", "#${A}+#${B}");
        assertEquals("1+2", result);
    }

    /**
     * metadata 中 value 为 null 时，替换为空串 ""，不抛 NPE
     */
    @Test
    public void testReplaceContentWithNullValueInMetadata() throws Exception {
        FileSystemFetcher fileSystemFetcher = new FileSystemFetcher();
        fileSystemFetcher.setPlaceholderResolver(ObjectBuilder.buildEmptyPlaceholderResolver());
        fileSystemFetcher.setPrefix("#");
        fileSystemFetcher.setArgs(null);
        WorkflowTask workflowTask = ObjectBuilder.buildWorkflowTask();
        workflowTask.getMetadata().put("${A}", "ok");
        workflowTask.getMetadata().put("${B}", null);
        workflowTask.getMetadata().put("${C}", "end");
        String result = fileSystemFetcher.replaceContent(workflowTask, "", "", "#${A}|#${B}|#${C}");
        assertEquals("ok||end", result);
    }

    /**
     * getMetadata 抛异常时返回原 content（不依赖 Mockito，使用 NettyRequest 匿名子类）
     */
    @Test
    public void testReplaceContentWhenGetMetadataThrows() throws Exception {
        FileSystemFetcher fileSystemFetcher = new FileSystemFetcher();
        fileSystemFetcher.setPlaceholderResolver(ObjectBuilder.buildEmptyPlaceholderResolver());
        fileSystemFetcher.setArgs(null);
        WorkflowTask workflowTask = new NettyRequest() {
            @Override
            public Map<String, Object> getMetadata() {
                throw new RuntimeException("mock getMetadata error");
            }
        };
        String content = "original content";
        String result = fileSystemFetcher.replaceContent(workflowTask, "", "", content);
        assertEquals(content, result);
    }

    @Test
    public void testUseAllowed() throws Exception {
        FileSystemFetcher fileSystemFetcher = new FileSystemFetcher();
        fileSystemFetcher.setPlaceholderResolver(ObjectBuilder.buildEmptyPlaceholderResolver());
        fileSystemFetcher.setDir(System.getProperty("user.dir") + "/src/test/resources/skills");
        fileSystemFetcher.setCached(true);
        fileSystemFetcher.setResourceService(ObjectBuilder.buildResourceService());
        fileSystemFetcher.setUsage("USAGE");
        fileSystemFetcher.setExpire(1000);
        fileSystemFetcher.init();
        List<SkillMetadata> skillMetadata = new ArrayList<>(fileSystemFetcher.fetchSkills(ObjectBuilder.buildWorkflowTask()).getSkills().values());
        assertEquals(Integer.valueOf(8), Integer.valueOf(skillMetadata.size()));
        String expect = "[{\"description\":\"Comprehensive spreadsheet creation, editing, and analysis with support for formulas, formatting, data analysis, and visualization. When Claude needs to work with spreadsheets (.xlsx, .xlsm, .csv, .tsv, etc) for: (1) Creating new spreadsheets with formulas and formatting, (2) Reading or analyzing data, (3) Modify existing spreadsheets while preserving formulas, (4) Data analysis and visualization in spreadsheets, or (5) Recalculating formulas\",\"name\":\"xlsx\"},{\"description\":\"Comprehensive PDF manipulation toolkit for extracting text and tables, creating new PDFs, merging/splitting documents, and handling forms. When Claude needs to fill in a PDF form or programmatically process, generate, or analyze PDF documents at scale.\",\"name\":\"pdf\"},{\"description\":\"Presentation creation, editing, and analysis. When Claude needs to work with presentations (.pptx files) for: (1) Creating new presentations, (2) Modifying or editing content, (3) Working with layouts, (4) Adding comments or speaker notes, or any other presentation tasks\",\"name\":\"pptx\"},{\"description\":\"Comprehensive document creation, editing, and analysis with support for tracked changes, comments, formatting preservation, and text extraction. When Claude needs to work with professional documents (.docx files) for: (1) Creating new documents, (2) Modifying or editing content, (3) Working with tracked changes, (4) Adding comments, or any other document tasks\",\"name\":\"docx\"},{\"description\":\"empty\",\"name\":\"empty\"},{\"compatibility\":\"Designed for Claude Code (or similar products)\",\"description\":\"Extract text and tables from PDF files, fill forms, merge documents.\",\"name\":\"pdf-processing\",\"metadata\":{\"xd\":\"example-org\",\"version\":\"1.0\"},\"allowed-tools\":[\"Bash(git:*)\",\"Bash(jq:*)\",\"Read\"]},{\"description\":\"Guide for creating effective skills. This skill should be used when users want to create a new skill (or update an existing skill) that extends Claude's capabilities with specialized knowledge, workflows, or tool integrations.\",\"name\":\"skill-creator\"},{\"description\":\"生成城市天际线的高质量图像，特别是包含地标建筑和特定光影效果的场景。\",\"name\":\"image-generation-city-skyline\",\"metadata\":{\"category\":\"image-generation\"}}]";
        assertEquals(expect, JsonUtils.write(skillMetadata));
    }

    @Test
    public void testUseAllowed1() throws Exception {
        FileSystemFetcher fileSystemFetcher = new FileSystemFetcher();
        fileSystemFetcher.setPlaceholderResolver(ObjectBuilder.buildEmptyPlaceholderResolver());
        fileSystemFetcher.setDir(System.getProperty("user.dir") + "/src/test/resources/skills");
        fileSystemFetcher.setResourceService(ObjectBuilder.buildResourceService());
        fileSystemFetcher.setUsage("USAGE");
        fileSystemFetcher.setExpire(1000);
        fileSystemFetcher.setCached(false);
        fileSystemFetcher.init();
        AllowedConfig allowedConfig = new AllowedConfig();
        allowedConfig.addBlack("pdf");
        List<SkillMetadata> skillMetadata = new ArrayList<>(fileSystemFetcher.fetchSkills(ObjectBuilder.buildWorkflowTask()).getSkills().values());
        assertEquals(Integer.valueOf(8), Integer.valueOf(skillMetadata.size()));
        String expect = "[{\"description\":\"Comprehensive spreadsheet creation, editing, and analysis with support for formulas, formatting, data analysis, and visualization. When Claude needs to work with spreadsheets (.xlsx, .xlsm, .csv, .tsv, etc) for: (1) Creating new spreadsheets with formulas and formatting, (2) Reading or analyzing data, (3) Modify existing spreadsheets while preserving formulas, (4) Data analysis and visualization in spreadsheets, or (5) Recalculating formulas\",\"name\":\"xlsx\"},{\"description\":\"Comprehensive PDF manipulation toolkit for extracting text and tables, creating new PDFs, merging/splitting documents, and handling forms. When Claude needs to fill in a PDF form or programmatically process, generate, or analyze PDF documents at scale.\",\"name\":\"pdf\"},{\"description\":\"Presentation creation, editing, and analysis. When Claude needs to work with presentations (.pptx files) for: (1) Creating new presentations, (2) Modifying or editing content, (3) Working with layouts, (4) Adding comments or speaker notes, or any other presentation tasks\",\"name\":\"pptx\"},{\"description\":\"Comprehensive document creation, editing, and analysis with support for tracked changes, comments, formatting preservation, and text extraction. When Claude needs to work with professional documents (.docx files) for: (1) Creating new documents, (2) Modifying or editing content, (3) Working with tracked changes, (4) Adding comments, or any other document tasks\",\"name\":\"docx\"},{\"description\":\"empty\",\"name\":\"empty\"},{\"compatibility\":\"Designed for Claude Code (or similar products)\",\"description\":\"Extract text and tables from PDF files, fill forms, merge documents.\",\"name\":\"pdf-processing\",\"metadata\":{\"xd\":\"example-org\",\"version\":\"1.0\"},\"allowed-tools\":[\"Bash(git:*)\",\"Bash(jq:*)\",\"Read\"]},{\"description\":\"Guide for creating effective skills. This skill should be used when users want to create a new skill (or update an existing skill) that extends Claude's capabilities with specialized knowledge, workflows, or tool integrations.\",\"name\":\"skill-creator\"},{\"description\":\"生成城市天际线的高质量图像，特别是包含地标建筑和特定光影效果的场景。\",\"name\":\"image-generation-city-skyline\",\"metadata\":{\"category\":\"image-generation\"}}]";
        assertEquals(expect, JsonUtils.write(skillMetadata));
    }

    @Test
    public void testUseAllowed2() throws Exception {
        FileSystemFetcher fileSystemFetcher = new FileSystemFetcher();
        fileSystemFetcher.setPlaceholderResolver(ObjectBuilder.buildEmptyPlaceholderResolver());
        fileSystemFetcher.setDir(System.getProperty("user.dir") + "/src/test/resources/skills");
        fileSystemFetcher.setResourceService(ObjectBuilder.buildResourceService());
        fileSystemFetcher.setUsage("USAGE");
        fileSystemFetcher.setExpire(1000);
        fileSystemFetcher.setCached(true);
        fileSystemFetcher.init();
        AllowedConfig allowedConfig = new AllowedConfig();
        allowedConfig.addWhite("pdf");
        List<SkillMetadata> skillMetadata = new ArrayList<>(fileSystemFetcher.fetchSkills(ObjectBuilder.buildWorkflowTask(), allowedConfig).getSkills().values());
        assertEquals(Integer.valueOf(1), Integer.valueOf(skillMetadata.size()));
        String expect = "[{\"description\":\"Comprehensive PDF manipulation toolkit for extracting text and tables, creating new PDFs, merging/splitting documents, and handling forms. When Claude needs to fill in a PDF form or programmatically process, generate, or analyze PDF documents at scale.\",\"name\":\"pdf\"}]";
        assertEquals(expect, JsonUtils.write(skillMetadata));
    }

    @Test
    public void testToolsException() throws Exception {
        Map<String, Object> skill = new HashMap<String, Object>() {

            @Override
            public Object get(Object key) {
                throw new RuntimeException();
            }
        };
        FileSystemFetcher.SkillVisitor visitor = new FileSystemFetcher.SkillVisitor(null);
        String[] result = visitor.tools(skill);
        Assert.assertNull(result);
    }

    @Test
    public void testMetadataException() throws Exception {
        Map<String, Object> skill = new HashMap<String, Object>() {

            @Override
            public Object get(Object key) {
                throw new RuntimeException();
            }
        };
        FileSystemFetcher.SkillVisitor visitor = new FileSystemFetcher.SkillVisitor(null);
        Map<String, Object> result = visitor.metadata(skill);
        Assert.assertNull(result);
    }

    @Test
    public void testResource() throws Exception {
        FileSystemFetcher fileSystemFetcher = new FileSystemFetcher();
        fileSystemFetcher.setDir(System.getProperty("user.dir") + "/src/test/resources/skills");
        fileSystemFetcher.setPlaceholderResolver(ObjectBuilder.buildEmptyPlaceholderResolver());
        fileSystemFetcher.setResourceService(ObjectBuilder.buildResourceService());
        fileSystemFetcher.setUsage("USAGE");
        fileSystemFetcher.setExpire(1000);
        fileSystemFetcher.setCached(true);
        fileSystemFetcher.init();
        fileSystemFetcher.fetchSkills(ObjectBuilder.buildWorkflowTask());
        String resource = fileSystemFetcher.fetchResource(ObjectBuilder.buildWorkflowTask(), "skill-creator", "LICENSE.txt");
        assertEquals("\n" +
                "The following is the beginning of resource `LICENSE.txt` in skill `skill-creator`\n" +
                "Test\n" +
                "The above is the full content of resource `LICENSE.txt` in skill `skill-creator`\n", resource);
    }

    @Test
    public void testResource2() throws Exception {
        FileSystemFetcher fileSystemFetcher = new FileSystemFetcher();
        fileSystemFetcher.setDir(System.getProperty("user.dir") + "/src/test/resources/skills");
        fileSystemFetcher.setPlaceholderResolver(ObjectBuilder.buildEmptyPlaceholderResolver());
        fileSystemFetcher.setResourceService(ObjectBuilder.buildResourceService());
        fileSystemFetcher.setUsage("USAGE");
        fileSystemFetcher.setExpire(1000);
        fileSystemFetcher.setCached(true);
        fileSystemFetcher.init();
        fileSystemFetcher.fetchSkills(ObjectBuilder.buildWorkflowTask());
        String resource = fileSystemFetcher.fetchResource(ObjectBuilder.buildWorkflowTask(), "skill-creator", "scripts/init_skill.py");
        assertEquals("\n" +
                "The following is the beginning of resource `scripts/init_skill.py` in skill `skill-creator`\n" +
                "1+1\n" +
                "The above is the full content of resource `scripts/init_skill.py` in skill `skill-creator`\n", resource);
    }

    @Test
    public void testResource3() throws Exception {
        FileSystemFetcher fileSystemFetcher = new FileSystemFetcher();
        fileSystemFetcher.setDir(System.getProperty("user.dir") + "/src/test/resources/skills");
        fileSystemFetcher.setPlaceholderResolver(ObjectBuilder.buildEmptyPlaceholderResolver());
        fileSystemFetcher.setResourceService(ObjectBuilder.buildResourceService());
        fileSystemFetcher.setUsage("USAGE");
        fileSystemFetcher.setExpire(1000);
        fileSystemFetcher.setCached(true);
        fileSystemFetcher.init();
        fileSystemFetcher.fetchSkills(ObjectBuilder.buildWorkflowTask());
        // NoSuchFileException
        Assert.assertTrue(fileSystemFetcher.fetchResource(ObjectBuilder.buildWorkflowTask(), "skill-creator", "scripts/init_skill2.py").startsWith("The skill resource failed to load"));
    }

    @Test
    public void testResource4() throws Exception {
        FileSystemFetcher fileSystemFetcher = new FileSystemFetcher();
        fileSystemFetcher.setDir(System.getProperty("user.dir") + "/src/test/resources/skills");
        fileSystemFetcher.setPlaceholderResolver(ObjectBuilder.buildEmptyPlaceholderResolver());
        fileSystemFetcher.setResourceService(ObjectBuilder.buildResourceService());
        fileSystemFetcher.setUsage("USAGE");
        fileSystemFetcher.setExpire(1000);
        fileSystemFetcher.setCached(true);
        fileSystemFetcher.init();
        fileSystemFetcher.fetchSkills(ObjectBuilder.buildWorkflowTask());
        // IllegalArgumentException
        Assert.assertTrue(fileSystemFetcher.fetchResource(ObjectBuilder.buildWorkflowTask(), "skill-creator2", "scripts/init_skill2.py").startsWith("The skill resource failed to load"));
    }

    @Test
    public void testFetchOverlap1() throws Exception {
        FileSystemFetcher fileSystemFetcher = new FileSystemFetcher();
        fileSystemFetcher.setDir(System.getProperty("user.dir") + "/src/test/resources/skills");
        fileSystemFetcher.setPlaceholderResolver(ObjectBuilder.buildEmptyPlaceholderResolver());
        fileSystemFetcher.setResourceService(ObjectBuilder.buildResourceService());
        fileSystemFetcher.setExpire(1000);
        fileSystemFetcher.setUsage("USAGE");
        fileSystemFetcher.setCached(false);
        fileSystemFetcher.init();
        fileSystemFetcher.fetchSkills(ObjectBuilder.buildWorkflowTask());
        String content = fileSystemFetcher.fetchResource(ObjectBuilder.buildWorkflowTask(), "skill-creator", "/SKILL.md");
        assertEquals("\n" +
                "The following is the beginning of resource `/SKILL.md` in skill `skill-creator`\n" +
                "# Skill Creator\n" +
                "\n" +
                "This skill provides guidance for creating effective skills.\n" +
                "\n" +
                "## About Skills\n" +
                "\n" +
                "Skills are modular, self-contained packages that extend Claude's capabilities by providing\n" +
                "specialized knowledge, workflows, and tools. Think of them as \"onboarding guides\" for specific\n" +
                "domains or tasks—they transform Claude from a general-purpose agent into a specialized agent\n" +
                "equipped with procedural knowledge that no model can fully possess.\n" +
                "\n" +
                "### What Skills Provide\n" +
                "\n" +
                "1. Specialized workflows - Multi-step procedures for specific domains\n" +
                "2. Tool integrations - Instructions for working with specific file formats or APIs\n" +
                "3. Domain expertise - Company-specific knowledge, schemas, business logic\n" +
                "4. Bundled resources - Scripts, references, and assets for complex and repetitive tasks\n" +
                "\n" +
                "### Anatomy of a Skill\n" +
                "\n" +
                "Every skill consists of a required SKILL.md file and optional bundled resources:\n" +
                "\n" +
                "```\n" +
                "skill-name/\n" +
                "├── SKILL.md (required)\n" +
                "│   ├── YAML frontmatter metadata (required)\n" +
                "│   │   ├── name: (required)\n" +
                "│   │   └── description: (required)\n" +
                "│   └── Markdown instructions (required)\n" +
                "└── Bundled Resources (optional)\n" +
                "    ├── scripts/          - Executable code (Python/Bash/etc.)\n" +
                "    ├── references/       - Documentation intended to be loaded into context as needed\n" +
                "    └── assets/           - Files used in output (templates, icons, fonts, etc.)\n" +
                "```\n" +
                "\n" +
                "#### SKILL.md (required)\n" +
                "\n" +
                "**Metadata Quality:** The `name` and `description` in YAML frontmatter determine when Claude will use the skill. Be specific about what the skill does and when to use it. Use the third-person (e.g. \"This skill should be used when...\" instead of \"Use this skill when...\").\n" +
                "\n" +
                "#### Bundled Resources (optional)\n" +
                "\n" +
                "##### Scripts (`scripts/`)\n" +
                "\n" +
                "Executable code (Python/Bash/etc.) for tasks that require deterministic reliability or are repeatedly rewritten.\n" +
                "\n" +
                "- **When to include**: When the same code is being rewritten repeatedly or deterministic reliability is needed\n" +
                "- **Example**: `scripts/rotate_pdf.py` for PDF rotation tasks\n" +
                "- **Benefits**: Token efficient, deterministic, may be executed without loading into context\n" +
                "- **Note**: Scripts may still need to be read by Claude for patching or environment-specific adjustments\n" +
                "\n" +
                "##### References (`references/`)\n" +
                "\n" +
                "Documentation and reference material intended to be loaded as needed into context to inform Claude's process and thinking.\n" +
                "\n" +
                "- **When to include**: For documentation that Claude should reference while working\n" +
                "- **Examples**: `references/finance.md` for financial schemas, `references/mnda.md` for company NDA template, `references/policies.md` for company policies, `references/api_docs.md` for API specifications\n" +
                "- **Use cases**: Database schemas, API documentation, domain knowledge, company policies, detailed workflow guides\n" +
                "- **Benefits**: Keeps SKILL.md lean, loaded only when Claude determines it's needed\n" +
                "- **Best practice**: If files are large (>10k words), include grep search patterns in SKILL.md\n" +
                "- **Avoid duplication**: Information should live in either SKILL.md or references files, not both. Prefer references files for detailed information unless it's truly core to the skill—this keeps SKILL.md lean while making information discoverable without hogging the context window. Keep only essential procedural instructions and workflow guidance in SKILL.md; move detailed reference material, schemas, and examples to references files.\n" +
                "\n" +
                "##### Assets (`assets/`)\n" +
                "\n" +
                "Files not intended to be loaded into context, but rather used within the output Claude produces.\n" +
                "\n" +
                "- **When to include**: When the skill needs files that will be used in the final output\n" +
                "- **Examples**: `assets/logo.png` for brand assets, `assets/slides.pptx` for PowerPoint templates, `assets/frontend-template/` for HTML/React boilerplate, `assets/font.ttf` for typography\n" +
                "- **Use cases**: Templates, images, icons, boilerplate code, fonts, sample documents that get copied or modified\n" +
                "- **Benefits**: Separates output resources from documentation, enables Claude to use files without loading them into context\n" +
                "\n" +
                "### Progressive Disclosure Design Principle\n" +
                "\n" +
                "Skills use a three-level loading system to manage context efficiently:\n" +
                "\n" +
                "1. **Metadata (name + description)** - Always in context (~100 words)\n" +
                "2. **SKILL.md body** - When skill triggers (<5k words)\n" +
                "3. **Bundled resources** - As needed by Claude (Unlimited*)\n" +
                "\n" +
                "*Unlimited because scripts can be executed without reading into context window.\n" +
                "\n" +
                "## Skill Creation Process\n" +
                "\n" +
                "To create a skill, follow the \"Skill Creation Process\" in order, skipping steps only if there is a clear reason why they are not applicable.\n" +
                "\n" +
                "### Step 1: Understanding the Skill with Concrete Examples\n" +
                "\n" +
                "Skip this step only when the skill's usage patterns are already clearly understood. It remains valuable even when working with an existing skill.\n" +
                "\n" +
                "To create an effective skill, clearly understand concrete examples of how the skill will be used. This understanding can come from either direct user examples or generated examples that are validated with user feedback.\n" +
                "\n" +
                "For example, when building an image-editor skill, relevant questions include:\n" +
                "\n" +
                "- \"What functionality should the image-editor skill support? Editing, rotating, anything else?\"\n" +
                "- \"Can you give some examples of how this skill would be used?\"\n" +
                "- \"I can imagine users asking for things like 'Remove the red-eye from this image' or 'Rotate this image'. Are there other ways you imagine this skill being used?\"\n" +
                "- \"What would a user say that should trigger this skill?\"\n" +
                "\n" +
                "To avoid overwhelming users, avoid asking too many questions in a single message. Start with the most important questions and follow up as needed for better effectiveness.\n" +
                "\n" +
                "Conclude this step when there is a clear sense of the functionality the skill should support.\n" +
                "\n" +
                "### Step 2: Planning the Reusable Skill Contents\n" +
                "\n" +
                "To turn concrete examples into an effective skill, analyze each example by:\n" +
                "\n" +
                "1. Considering how to execute on the example from scratch\n" +
                "2. Identifying what scripts, references, and assets would be helpful when executing these workflows repeatedly\n" +
                "\n" +
                "Example: When building a `pdf-editor` skill to handle queries like \"Help me rotate this PDF,\" the analysis shows:\n" +
                "\n" +
                "1. Rotating a PDF requires re-writing the same code each time\n" +
                "2. A `scripts/rotate_pdf.py` script would be helpful to store in the skill\n" +
                "\n" +
                "Example: When designing a `frontend-webapp-builder` skill for queries like \"Build me a todo app\" or \"Build me a dashboard to track my steps,\" the analysis shows:\n" +
                "\n" +
                "1. Writing a frontend webapp requires the same boilerplate HTML/React each time\n" +
                "2. An `assets/hello-world/` template containing the boilerplate HTML/React project files would be helpful to store in the skill\n" +
                "\n" +
                "Example: When building a `big-query` skill to handle queries like \"How many users have logged in today?\" the analysis shows:\n" +
                "\n" +
                "1. Querying BigQuery requires re-discovering the table schemas and relationships each time\n" +
                "2. A `references/schema.md` file documenting the table schemas would be helpful to store in the skill\n" +
                "\n" +
                "To establish the skill's contents, analyze each concrete example to create a list of the reusable resources to include: scripts, references, and assets.\n" +
                "\n" +
                "### Step 3: Initializing the Skill\n" +
                "\n" +
                "At this point, it is time to actually create the skill.\n" +
                "\n" +
                "Skip this step only if the skill being developed already exists, and iteration or packaging is needed. In this case, continue to the next step.\n" +
                "\n" +
                "When creating a new skill from scratch, always run the `init_skill.py` script. The script conveniently generates a new template skill directory that automatically includes everything a skill requires, making the skill creation process much more efficient and reliable.\n" +
                "\n" +
                "Usage:\n" +
                "\n" +
                "```bash\n" +
                "scripts/init_skill.py <skill-name> --path <output-directory>\n" +
                "```\n" +
                "\n" +
                "The script:\n" +
                "\n" +
                "- Creates the skill directory at the specified path\n" +
                "- Generates a SKILL.md template with proper frontmatter and TODO placeholders\n" +
                "- Creates example resource directories: `scripts/`, `references/`, and `assets/`\n" +
                "- Adds example files in each directory that can be customized or deleted\n" +
                "\n" +
                "After initialization, customize or remove the generated SKILL.md and example files as needed.\n" +
                "\n" +
                "### Step 4: Edit the Skill\n" +
                "\n" +
                "When editing the (newly-generated or existing) skill, remember that the skill is being created for another instance of Claude to use. Focus on including information that would be beneficial and non-obvious to Claude. Consider what procedural knowledge, domain-specific details, or reusable assets would help another Claude instance execute these tasks more effectively.\n" +
                "\n" +
                "#### Start with Reusable Skill Contents\n" +
                "\n" +
                "To begin implementation, start with the reusable resources identified above: `scripts/`, `references/`, and `assets/` files. Note that this step may require user input. For example, when implementing a `brand-guidelines` skill, the user may need to provide brand assets or templates to store in `assets/`, or documentation to store in `references/`.\n" +
                "\n" +
                "Also, delete any example files and directories not needed for the skill. The initialization script creates example files in `scripts/`, `references/`, and `assets/` to demonstrate structure, but most skills won't need all of them.\n" +
                "\n" +
                "#### Update SKILL.md\n" +
                "\n" +
                "**Writing Style:** Write the entire skill using **imperative/infinitive form** (verb-first instructions), not second person. Use objective, instructional language (e.g., \"To accomplish X, do Y\" rather than \"You should do X\" or \"If you need to do X\"). This maintains consistency and clarity for AI consumption.\n" +
                "\n" +
                "To complete SKILL.md, answer the following questions:\n" +
                "\n" +
                "1. What is the purpose of the skill, in a few sentences?\n" +
                "2. When should the skill be used?\n" +
                "3. In practice, how should Claude use the skill? All reusable skill contents developed above should be referenced so that Claude knows how to use them.\n" +
                "\n" +
                "### Step 5: Packaging a Skill\n" +
                "\n" +
                "Once the skill is ready, it should be packaged into a distributable zip file that gets shared with the user. The packaging process automatically validates the skill first to ensure it meets all requirements:\n" +
                "\n" +
                "```bash\n" +
                "scripts/package_skill.py <path/to/skill-folder>\n" +
                "```\n" +
                "\n" +
                "Optional output directory specification:\n" +
                "\n" +
                "```bash\n" +
                "scripts/package_skill.py <path/to/skill-folder> ./dist\n" +
                "```\n" +
                "\n" +
                "The packaging script will:\n" +
                "\n" +
                "1. **Validate** the skill automatically, checking:\n" +
                "   - YAML frontmatter format and required fields\n" +
                "   - Skill naming conventions and directory structure\n" +
                "   - Description completeness and quality\n" +
                "   - File organization and resource references\n" +
                "\n" +
                "2. **Package** the skill if validation passes, creating a zip file named after the skill (e.g., `my-skill.zip`) that includes all files and maintains the proper directory structure for distribution.\n" +
                "\n" +
                "If validation fails, the script will report the errors and exit without creating a package. Fix any validation errors and run the packaging command again.\n" +
                "\n" +
                "### Step 6: Iterate\n" +
                "\n" +
                "After testing the skill, users may request improvements. Often this happens right after using the skill, with fresh context of how the skill performed.\n" +
                "\n" +
                "**Iteration workflow:**\n" +
                "1. Use the skill on real tasks\n" +
                "2. Notice struggles or inefficiencies\n" +
                "3. Identify how SKILL.md or bundled resources should be updated\n" +
                "4. Implement changes and test again\n" +
                "The above is the full content of resource `/SKILL.md` in skill `skill-creator`\n", content);
    }

    @Test
    public void testFetchOverlap2() throws Exception {
        FileSystemFetcher fileSystemFetcher = new FileSystemFetcher();
        fileSystemFetcher.setDir(System.getProperty("user.dir") + "/src/test/resources/skills");
        fileSystemFetcher.setPlaceholderResolver(ObjectBuilder.buildEmptyPlaceholderResolver());
        fileSystemFetcher.setResourceService(ObjectBuilder.buildResourceService());
        fileSystemFetcher.setExpire(1000);
        fileSystemFetcher.setUsage("USAGE");
        fileSystemFetcher.setCached(false);
        fileSystemFetcher.init();
        fileSystemFetcher.fetchSkills(ObjectBuilder.buildWorkflowTask());
        String content = fileSystemFetcher.fetchResource(ObjectBuilder.buildWorkflowTask(), "skill-creator", "skill-creator/SKILL.md");
        assertEquals("\n" +
                "The following is the beginning of resource `skill-creator/SKILL.md` in skill `skill-creator`\n" +
                "# Skill Creator\n" +
                "\n" +
                "This skill provides guidance for creating effective skills.\n" +
                "\n" +
                "## About Skills\n" +
                "\n" +
                "Skills are modular, self-contained packages that extend Claude's capabilities by providing\n" +
                "specialized knowledge, workflows, and tools. Think of them as \"onboarding guides\" for specific\n" +
                "domains or tasks—they transform Claude from a general-purpose agent into a specialized agent\n" +
                "equipped with procedural knowledge that no model can fully possess.\n" +
                "\n" +
                "### What Skills Provide\n" +
                "\n" +
                "1. Specialized workflows - Multi-step procedures for specific domains\n" +
                "2. Tool integrations - Instructions for working with specific file formats or APIs\n" +
                "3. Domain expertise - Company-specific knowledge, schemas, business logic\n" +
                "4. Bundled resources - Scripts, references, and assets for complex and repetitive tasks\n" +
                "\n" +
                "### Anatomy of a Skill\n" +
                "\n" +
                "Every skill consists of a required SKILL.md file and optional bundled resources:\n" +
                "\n" +
                "```\n" +
                "skill-name/\n" +
                "├── SKILL.md (required)\n" +
                "│   ├── YAML frontmatter metadata (required)\n" +
                "│   │   ├── name: (required)\n" +
                "│   │   └── description: (required)\n" +
                "│   └── Markdown instructions (required)\n" +
                "└── Bundled Resources (optional)\n" +
                "    ├── scripts/          - Executable code (Python/Bash/etc.)\n" +
                "    ├── references/       - Documentation intended to be loaded into context as needed\n" +
                "    └── assets/           - Files used in output (templates, icons, fonts, etc.)\n" +
                "```\n" +
                "\n" +
                "#### SKILL.md (required)\n" +
                "\n" +
                "**Metadata Quality:** The `name` and `description` in YAML frontmatter determine when Claude will use the skill. Be specific about what the skill does and when to use it. Use the third-person (e.g. \"This skill should be used when...\" instead of \"Use this skill when...\").\n" +
                "\n" +
                "#### Bundled Resources (optional)\n" +
                "\n" +
                "##### Scripts (`scripts/`)\n" +
                "\n" +
                "Executable code (Python/Bash/etc.) for tasks that require deterministic reliability or are repeatedly rewritten.\n" +
                "\n" +
                "- **When to include**: When the same code is being rewritten repeatedly or deterministic reliability is needed\n" +
                "- **Example**: `scripts/rotate_pdf.py` for PDF rotation tasks\n" +
                "- **Benefits**: Token efficient, deterministic, may be executed without loading into context\n" +
                "- **Note**: Scripts may still need to be read by Claude for patching or environment-specific adjustments\n" +
                "\n" +
                "##### References (`references/`)\n" +
                "\n" +
                "Documentation and reference material intended to be loaded as needed into context to inform Claude's process and thinking.\n" +
                "\n" +
                "- **When to include**: For documentation that Claude should reference while working\n" +
                "- **Examples**: `references/finance.md` for financial schemas, `references/mnda.md` for company NDA template, `references/policies.md` for company policies, `references/api_docs.md` for API specifications\n" +
                "- **Use cases**: Database schemas, API documentation, domain knowledge, company policies, detailed workflow guides\n" +
                "- **Benefits**: Keeps SKILL.md lean, loaded only when Claude determines it's needed\n" +
                "- **Best practice**: If files are large (>10k words), include grep search patterns in SKILL.md\n" +
                "- **Avoid duplication**: Information should live in either SKILL.md or references files, not both. Prefer references files for detailed information unless it's truly core to the skill—this keeps SKILL.md lean while making information discoverable without hogging the context window. Keep only essential procedural instructions and workflow guidance in SKILL.md; move detailed reference material, schemas, and examples to references files.\n" +
                "\n" +
                "##### Assets (`assets/`)\n" +
                "\n" +
                "Files not intended to be loaded into context, but rather used within the output Claude produces.\n" +
                "\n" +
                "- **When to include**: When the skill needs files that will be used in the final output\n" +
                "- **Examples**: `assets/logo.png` for brand assets, `assets/slides.pptx` for PowerPoint templates, `assets/frontend-template/` for HTML/React boilerplate, `assets/font.ttf` for typography\n" +
                "- **Use cases**: Templates, images, icons, boilerplate code, fonts, sample documents that get copied or modified\n" +
                "- **Benefits**: Separates output resources from documentation, enables Claude to use files without loading them into context\n" +
                "\n" +
                "### Progressive Disclosure Design Principle\n" +
                "\n" +
                "Skills use a three-level loading system to manage context efficiently:\n" +
                "\n" +
                "1. **Metadata (name + description)** - Always in context (~100 words)\n" +
                "2. **SKILL.md body** - When skill triggers (<5k words)\n" +
                "3. **Bundled resources** - As needed by Claude (Unlimited*)\n" +
                "\n" +
                "*Unlimited because scripts can be executed without reading into context window.\n" +
                "\n" +
                "## Skill Creation Process\n" +
                "\n" +
                "To create a skill, follow the \"Skill Creation Process\" in order, skipping steps only if there is a clear reason why they are not applicable.\n" +
                "\n" +
                "### Step 1: Understanding the Skill with Concrete Examples\n" +
                "\n" +
                "Skip this step only when the skill's usage patterns are already clearly understood. It remains valuable even when working with an existing skill.\n" +
                "\n" +
                "To create an effective skill, clearly understand concrete examples of how the skill will be used. This understanding can come from either direct user examples or generated examples that are validated with user feedback.\n" +
                "\n" +
                "For example, when building an image-editor skill, relevant questions include:\n" +
                "\n" +
                "- \"What functionality should the image-editor skill support? Editing, rotating, anything else?\"\n" +
                "- \"Can you give some examples of how this skill would be used?\"\n" +
                "- \"I can imagine users asking for things like 'Remove the red-eye from this image' or 'Rotate this image'. Are there other ways you imagine this skill being used?\"\n" +
                "- \"What would a user say that should trigger this skill?\"\n" +
                "\n" +
                "To avoid overwhelming users, avoid asking too many questions in a single message. Start with the most important questions and follow up as needed for better effectiveness.\n" +
                "\n" +
                "Conclude this step when there is a clear sense of the functionality the skill should support.\n" +
                "\n" +
                "### Step 2: Planning the Reusable Skill Contents\n" +
                "\n" +
                "To turn concrete examples into an effective skill, analyze each example by:\n" +
                "\n" +
                "1. Considering how to execute on the example from scratch\n" +
                "2. Identifying what scripts, references, and assets would be helpful when executing these workflows repeatedly\n" +
                "\n" +
                "Example: When building a `pdf-editor` skill to handle queries like \"Help me rotate this PDF,\" the analysis shows:\n" +
                "\n" +
                "1. Rotating a PDF requires re-writing the same code each time\n" +
                "2. A `scripts/rotate_pdf.py` script would be helpful to store in the skill\n" +
                "\n" +
                "Example: When designing a `frontend-webapp-builder` skill for queries like \"Build me a todo app\" or \"Build me a dashboard to track my steps,\" the analysis shows:\n" +
                "\n" +
                "1. Writing a frontend webapp requires the same boilerplate HTML/React each time\n" +
                "2. An `assets/hello-world/` template containing the boilerplate HTML/React project files would be helpful to store in the skill\n" +
                "\n" +
                "Example: When building a `big-query` skill to handle queries like \"How many users have logged in today?\" the analysis shows:\n" +
                "\n" +
                "1. Querying BigQuery requires re-discovering the table schemas and relationships each time\n" +
                "2. A `references/schema.md` file documenting the table schemas would be helpful to store in the skill\n" +
                "\n" +
                "To establish the skill's contents, analyze each concrete example to create a list of the reusable resources to include: scripts, references, and assets.\n" +
                "\n" +
                "### Step 3: Initializing the Skill\n" +
                "\n" +
                "At this point, it is time to actually create the skill.\n" +
                "\n" +
                "Skip this step only if the skill being developed already exists, and iteration or packaging is needed. In this case, continue to the next step.\n" +
                "\n" +
                "When creating a new skill from scratch, always run the `init_skill.py` script. The script conveniently generates a new template skill directory that automatically includes everything a skill requires, making the skill creation process much more efficient and reliable.\n" +
                "\n" +
                "Usage:\n" +
                "\n" +
                "```bash\n" +
                "scripts/init_skill.py <skill-name> --path <output-directory>\n" +
                "```\n" +
                "\n" +
                "The script:\n" +
                "\n" +
                "- Creates the skill directory at the specified path\n" +
                "- Generates a SKILL.md template with proper frontmatter and TODO placeholders\n" +
                "- Creates example resource directories: `scripts/`, `references/`, and `assets/`\n" +
                "- Adds example files in each directory that can be customized or deleted\n" +
                "\n" +
                "After initialization, customize or remove the generated SKILL.md and example files as needed.\n" +
                "\n" +
                "### Step 4: Edit the Skill\n" +
                "\n" +
                "When editing the (newly-generated or existing) skill, remember that the skill is being created for another instance of Claude to use. Focus on including information that would be beneficial and non-obvious to Claude. Consider what procedural knowledge, domain-specific details, or reusable assets would help another Claude instance execute these tasks more effectively.\n" +
                "\n" +
                "#### Start with Reusable Skill Contents\n" +
                "\n" +
                "To begin implementation, start with the reusable resources identified above: `scripts/`, `references/`, and `assets/` files. Note that this step may require user input. For example, when implementing a `brand-guidelines` skill, the user may need to provide brand assets or templates to store in `assets/`, or documentation to store in `references/`.\n" +
                "\n" +
                "Also, delete any example files and directories not needed for the skill. The initialization script creates example files in `scripts/`, `references/`, and `assets/` to demonstrate structure, but most skills won't need all of them.\n" +
                "\n" +
                "#### Update SKILL.md\n" +
                "\n" +
                "**Writing Style:** Write the entire skill using **imperative/infinitive form** (verb-first instructions), not second person. Use objective, instructional language (e.g., \"To accomplish X, do Y\" rather than \"You should do X\" or \"If you need to do X\"). This maintains consistency and clarity for AI consumption.\n" +
                "\n" +
                "To complete SKILL.md, answer the following questions:\n" +
                "\n" +
                "1. What is the purpose of the skill, in a few sentences?\n" +
                "2. When should the skill be used?\n" +
                "3. In practice, how should Claude use the skill? All reusable skill contents developed above should be referenced so that Claude knows how to use them.\n" +
                "\n" +
                "### Step 5: Packaging a Skill\n" +
                "\n" +
                "Once the skill is ready, it should be packaged into a distributable zip file that gets shared with the user. The packaging process automatically validates the skill first to ensure it meets all requirements:\n" +
                "\n" +
                "```bash\n" +
                "scripts/package_skill.py <path/to/skill-folder>\n" +
                "```\n" +
                "\n" +
                "Optional output directory specification:\n" +
                "\n" +
                "```bash\n" +
                "scripts/package_skill.py <path/to/skill-folder> ./dist\n" +
                "```\n" +
                "\n" +
                "The packaging script will:\n" +
                "\n" +
                "1. **Validate** the skill automatically, checking:\n" +
                "   - YAML frontmatter format and required fields\n" +
                "   - Skill naming conventions and directory structure\n" +
                "   - Description completeness and quality\n" +
                "   - File organization and resource references\n" +
                "\n" +
                "2. **Package** the skill if validation passes, creating a zip file named after the skill (e.g., `my-skill.zip`) that includes all files and maintains the proper directory structure for distribution.\n" +
                "\n" +
                "If validation fails, the script will report the errors and exit without creating a package. Fix any validation errors and run the packaging command again.\n" +
                "\n" +
                "### Step 6: Iterate\n" +
                "\n" +
                "After testing the skill, users may request improvements. Often this happens right after using the skill, with fresh context of how the skill performed.\n" +
                "\n" +
                "**Iteration workflow:**\n" +
                "1. Use the skill on real tasks\n" +
                "2. Notice struggles or inefficiencies\n" +
                "3. Identify how SKILL.md or bundled resources should be updated\n" +
                "4. Implement changes and test again\n" +
                "The above is the full content of resource `skill-creator/SKILL.md` in skill `skill-creator`\n", content);
    }

    @Test
    public void testFetchOverlap3() throws Exception {
        FileSystemFetcher fileSystemFetcher = new FileSystemFetcher();
        fileSystemFetcher.setDir(System.getProperty("user.dir") + "/src/test/resources/skills");
        fileSystemFetcher.setPlaceholderResolver(ObjectBuilder.buildEmptyPlaceholderResolver());
        fileSystemFetcher.setResourceService(ObjectBuilder.buildResourceService());
        fileSystemFetcher.setExpire(1000);
        fileSystemFetcher.setUsage("USAGE");
        fileSystemFetcher.setCached(false);
        fileSystemFetcher.init();
        fileSystemFetcher.fetchSkills(ObjectBuilder.buildWorkflowTask());
        String content = fileSystemFetcher.fetchResource(ObjectBuilder.buildWorkflowTask(), "skill-creator", "/skill-creator/SKILL.md");
        assertEquals("\n" +
                "The following is the beginning of resource `/skill-creator/SKILL.md` in skill `skill-creator`\n" +
                "# Skill Creator\n" +
                "\n" +
                "This skill provides guidance for creating effective skills.\n" +
                "\n" +
                "## About Skills\n" +
                "\n" +
                "Skills are modular, self-contained packages that extend Claude's capabilities by providing\n" +
                "specialized knowledge, workflows, and tools. Think of them as \"onboarding guides\" for specific\n" +
                "domains or tasks—they transform Claude from a general-purpose agent into a specialized agent\n" +
                "equipped with procedural knowledge that no model can fully possess.\n" +
                "\n" +
                "### What Skills Provide\n" +
                "\n" +
                "1. Specialized workflows - Multi-step procedures for specific domains\n" +
                "2. Tool integrations - Instructions for working with specific file formats or APIs\n" +
                "3. Domain expertise - Company-specific knowledge, schemas, business logic\n" +
                "4. Bundled resources - Scripts, references, and assets for complex and repetitive tasks\n" +
                "\n" +
                "### Anatomy of a Skill\n" +
                "\n" +
                "Every skill consists of a required SKILL.md file and optional bundled resources:\n" +
                "\n" +
                "```\n" +
                "skill-name/\n" +
                "├── SKILL.md (required)\n" +
                "│   ├── YAML frontmatter metadata (required)\n" +
                "│   │   ├── name: (required)\n" +
                "│   │   └── description: (required)\n" +
                "│   └── Markdown instructions (required)\n" +
                "└── Bundled Resources (optional)\n" +
                "    ├── scripts/          - Executable code (Python/Bash/etc.)\n" +
                "    ├── references/       - Documentation intended to be loaded into context as needed\n" +
                "    └── assets/           - Files used in output (templates, icons, fonts, etc.)\n" +
                "```\n" +
                "\n" +
                "#### SKILL.md (required)\n" +
                "\n" +
                "**Metadata Quality:** The `name` and `description` in YAML frontmatter determine when Claude will use the skill. Be specific about what the skill does and when to use it. Use the third-person (e.g. \"This skill should be used when...\" instead of \"Use this skill when...\").\n" +
                "\n" +
                "#### Bundled Resources (optional)\n" +
                "\n" +
                "##### Scripts (`scripts/`)\n" +
                "\n" +
                "Executable code (Python/Bash/etc.) for tasks that require deterministic reliability or are repeatedly rewritten.\n" +
                "\n" +
                "- **When to include**: When the same code is being rewritten repeatedly or deterministic reliability is needed\n" +
                "- **Example**: `scripts/rotate_pdf.py` for PDF rotation tasks\n" +
                "- **Benefits**: Token efficient, deterministic, may be executed without loading into context\n" +
                "- **Note**: Scripts may still need to be read by Claude for patching or environment-specific adjustments\n" +
                "\n" +
                "##### References (`references/`)\n" +
                "\n" +
                "Documentation and reference material intended to be loaded as needed into context to inform Claude's process and thinking.\n" +
                "\n" +
                "- **When to include**: For documentation that Claude should reference while working\n" +
                "- **Examples**: `references/finance.md` for financial schemas, `references/mnda.md` for company NDA template, `references/policies.md` for company policies, `references/api_docs.md` for API specifications\n" +
                "- **Use cases**: Database schemas, API documentation, domain knowledge, company policies, detailed workflow guides\n" +
                "- **Benefits**: Keeps SKILL.md lean, loaded only when Claude determines it's needed\n" +
                "- **Best practice**: If files are large (>10k words), include grep search patterns in SKILL.md\n" +
                "- **Avoid duplication**: Information should live in either SKILL.md or references files, not both. Prefer references files for detailed information unless it's truly core to the skill—this keeps SKILL.md lean while making information discoverable without hogging the context window. Keep only essential procedural instructions and workflow guidance in SKILL.md; move detailed reference material, schemas, and examples to references files.\n" +
                "\n" +
                "##### Assets (`assets/`)\n" +
                "\n" +
                "Files not intended to be loaded into context, but rather used within the output Claude produces.\n" +
                "\n" +
                "- **When to include**: When the skill needs files that will be used in the final output\n" +
                "- **Examples**: `assets/logo.png` for brand assets, `assets/slides.pptx` for PowerPoint templates, `assets/frontend-template/` for HTML/React boilerplate, `assets/font.ttf` for typography\n" +
                "- **Use cases**: Templates, images, icons, boilerplate code, fonts, sample documents that get copied or modified\n" +
                "- **Benefits**: Separates output resources from documentation, enables Claude to use files without loading them into context\n" +
                "\n" +
                "### Progressive Disclosure Design Principle\n" +
                "\n" +
                "Skills use a three-level loading system to manage context efficiently:\n" +
                "\n" +
                "1. **Metadata (name + description)** - Always in context (~100 words)\n" +
                "2. **SKILL.md body** - When skill triggers (<5k words)\n" +
                "3. **Bundled resources** - As needed by Claude (Unlimited*)\n" +
                "\n" +
                "*Unlimited because scripts can be executed without reading into context window.\n" +
                "\n" +
                "## Skill Creation Process\n" +
                "\n" +
                "To create a skill, follow the \"Skill Creation Process\" in order, skipping steps only if there is a clear reason why they are not applicable.\n" +
                "\n" +
                "### Step 1: Understanding the Skill with Concrete Examples\n" +
                "\n" +
                "Skip this step only when the skill's usage patterns are already clearly understood. It remains valuable even when working with an existing skill.\n" +
                "\n" +
                "To create an effective skill, clearly understand concrete examples of how the skill will be used. This understanding can come from either direct user examples or generated examples that are validated with user feedback.\n" +
                "\n" +
                "For example, when building an image-editor skill, relevant questions include:\n" +
                "\n" +
                "- \"What functionality should the image-editor skill support? Editing, rotating, anything else?\"\n" +
                "- \"Can you give some examples of how this skill would be used?\"\n" +
                "- \"I can imagine users asking for things like 'Remove the red-eye from this image' or 'Rotate this image'. Are there other ways you imagine this skill being used?\"\n" +
                "- \"What would a user say that should trigger this skill?\"\n" +
                "\n" +
                "To avoid overwhelming users, avoid asking too many questions in a single message. Start with the most important questions and follow up as needed for better effectiveness.\n" +
                "\n" +
                "Conclude this step when there is a clear sense of the functionality the skill should support.\n" +
                "\n" +
                "### Step 2: Planning the Reusable Skill Contents\n" +
                "\n" +
                "To turn concrete examples into an effective skill, analyze each example by:\n" +
                "\n" +
                "1. Considering how to execute on the example from scratch\n" +
                "2. Identifying what scripts, references, and assets would be helpful when executing these workflows repeatedly\n" +
                "\n" +
                "Example: When building a `pdf-editor` skill to handle queries like \"Help me rotate this PDF,\" the analysis shows:\n" +
                "\n" +
                "1. Rotating a PDF requires re-writing the same code each time\n" +
                "2. A `scripts/rotate_pdf.py` script would be helpful to store in the skill\n" +
                "\n" +
                "Example: When designing a `frontend-webapp-builder` skill for queries like \"Build me a todo app\" or \"Build me a dashboard to track my steps,\" the analysis shows:\n" +
                "\n" +
                "1. Writing a frontend webapp requires the same boilerplate HTML/React each time\n" +
                "2. An `assets/hello-world/` template containing the boilerplate HTML/React project files would be helpful to store in the skill\n" +
                "\n" +
                "Example: When building a `big-query` skill to handle queries like \"How many users have logged in today?\" the analysis shows:\n" +
                "\n" +
                "1. Querying BigQuery requires re-discovering the table schemas and relationships each time\n" +
                "2. A `references/schema.md` file documenting the table schemas would be helpful to store in the skill\n" +
                "\n" +
                "To establish the skill's contents, analyze each concrete example to create a list of the reusable resources to include: scripts, references, and assets.\n" +
                "\n" +
                "### Step 3: Initializing the Skill\n" +
                "\n" +
                "At this point, it is time to actually create the skill.\n" +
                "\n" +
                "Skip this step only if the skill being developed already exists, and iteration or packaging is needed. In this case, continue to the next step.\n" +
                "\n" +
                "When creating a new skill from scratch, always run the `init_skill.py` script. The script conveniently generates a new template skill directory that automatically includes everything a skill requires, making the skill creation process much more efficient and reliable.\n" +
                "\n" +
                "Usage:\n" +
                "\n" +
                "```bash\n" +
                "scripts/init_skill.py <skill-name> --path <output-directory>\n" +
                "```\n" +
                "\n" +
                "The script:\n" +
                "\n" +
                "- Creates the skill directory at the specified path\n" +
                "- Generates a SKILL.md template with proper frontmatter and TODO placeholders\n" +
                "- Creates example resource directories: `scripts/`, `references/`, and `assets/`\n" +
                "- Adds example files in each directory that can be customized or deleted\n" +
                "\n" +
                "After initialization, customize or remove the generated SKILL.md and example files as needed.\n" +
                "\n" +
                "### Step 4: Edit the Skill\n" +
                "\n" +
                "When editing the (newly-generated or existing) skill, remember that the skill is being created for another instance of Claude to use. Focus on including information that would be beneficial and non-obvious to Claude. Consider what procedural knowledge, domain-specific details, or reusable assets would help another Claude instance execute these tasks more effectively.\n" +
                "\n" +
                "#### Start with Reusable Skill Contents\n" +
                "\n" +
                "To begin implementation, start with the reusable resources identified above: `scripts/`, `references/`, and `assets/` files. Note that this step may require user input. For example, when implementing a `brand-guidelines` skill, the user may need to provide brand assets or templates to store in `assets/`, or documentation to store in `references/`.\n" +
                "\n" +
                "Also, delete any example files and directories not needed for the skill. The initialization script creates example files in `scripts/`, `references/`, and `assets/` to demonstrate structure, but most skills won't need all of them.\n" +
                "\n" +
                "#### Update SKILL.md\n" +
                "\n" +
                "**Writing Style:** Write the entire skill using **imperative/infinitive form** (verb-first instructions), not second person. Use objective, instructional language (e.g., \"To accomplish X, do Y\" rather than \"You should do X\" or \"If you need to do X\"). This maintains consistency and clarity for AI consumption.\n" +
                "\n" +
                "To complete SKILL.md, answer the following questions:\n" +
                "\n" +
                "1. What is the purpose of the skill, in a few sentences?\n" +
                "2. When should the skill be used?\n" +
                "3. In practice, how should Claude use the skill? All reusable skill contents developed above should be referenced so that Claude knows how to use them.\n" +
                "\n" +
                "### Step 5: Packaging a Skill\n" +
                "\n" +
                "Once the skill is ready, it should be packaged into a distributable zip file that gets shared with the user. The packaging process automatically validates the skill first to ensure it meets all requirements:\n" +
                "\n" +
                "```bash\n" +
                "scripts/package_skill.py <path/to/skill-folder>\n" +
                "```\n" +
                "\n" +
                "Optional output directory specification:\n" +
                "\n" +
                "```bash\n" +
                "scripts/package_skill.py <path/to/skill-folder> ./dist\n" +
                "```\n" +
                "\n" +
                "The packaging script will:\n" +
                "\n" +
                "1. **Validate** the skill automatically, checking:\n" +
                "   - YAML frontmatter format and required fields\n" +
                "   - Skill naming conventions and directory structure\n" +
                "   - Description completeness and quality\n" +
                "   - File organization and resource references\n" +
                "\n" +
                "2. **Package** the skill if validation passes, creating a zip file named after the skill (e.g., `my-skill.zip`) that includes all files and maintains the proper directory structure for distribution.\n" +
                "\n" +
                "If validation fails, the script will report the errors and exit without creating a package. Fix any validation errors and run the packaging command again.\n" +
                "\n" +
                "### Step 6: Iterate\n" +
                "\n" +
                "After testing the skill, users may request improvements. Often this happens right after using the skill, with fresh context of how the skill performed.\n" +
                "\n" +
                "**Iteration workflow:**\n" +
                "1. Use the skill on real tasks\n" +
                "2. Notice struggles or inefficiencies\n" +
                "3. Identify how SKILL.md or bundled resources should be updated\n" +
                "4. Implement changes and test again\n" +
                "The above is the full content of resource `/skill-creator/SKILL.md` in skill `skill-creator`\n", content);
    }

    @Test
    public void testFetchOverlap4() throws Exception {
        FileSystemFetcher fileSystemFetcher = new FileSystemFetcher();
        fileSystemFetcher.setDir(System.getProperty("user.dir") + "/src/test/resources/skills");
        fileSystemFetcher.setPlaceholderResolver(ObjectBuilder.buildEmptyPlaceholderResolver());
        fileSystemFetcher.setResourceService(ObjectBuilder.buildResourceService());
        fileSystemFetcher.setUsage("USAGE");
        fileSystemFetcher.setExpire(100000);
        fileSystemFetcher.setCached(true);
        fileSystemFetcher.init();
        fileSystemFetcher.fetchSkills(ObjectBuilder.buildWorkflowTask());
        String content = fileSystemFetcher.fetchResource(ObjectBuilder.buildWorkflowTask(), "skill-creator", ".md");
        assertEquals("\n" +
                "The following is the beginning of resource `.md` in skill `skill-creator`\n" +
                "# Skill Creator\n" +
                "\n" +
                "This skill provides guidance for creating effective skills.\n" +
                "\n" +
                "## About Skills\n" +
                "\n" +
                "Skills are modular, self-contained packages that extend Claude's capabilities by providing\n" +
                "specialized knowledge, workflows, and tools. Think of them as \"onboarding guides\" for specific\n" +
                "domains or tasks—they transform Claude from a general-purpose agent into a specialized agent\n" +
                "equipped with procedural knowledge that no model can fully possess.\n" +
                "\n" +
                "### What Skills Provide\n" +
                "\n" +
                "1. Specialized workflows - Multi-step procedures for specific domains\n" +
                "2. Tool integrations - Instructions for working with specific file formats or APIs\n" +
                "3. Domain expertise - Company-specific knowledge, schemas, business logic\n" +
                "4. Bundled resources - Scripts, references, and assets for complex and repetitive tasks\n" +
                "\n" +
                "### Anatomy of a Skill\n" +
                "\n" +
                "Every skill consists of a required SKILL.md file and optional bundled resources:\n" +
                "\n" +
                "```\n" +
                "skill-name/\n" +
                "├── SKILL.md (required)\n" +
                "│   ├── YAML frontmatter metadata (required)\n" +
                "│   │   ├── name: (required)\n" +
                "│   │   └── description: (required)\n" +
                "│   └── Markdown instructions (required)\n" +
                "└── Bundled Resources (optional)\n" +
                "    ├── scripts/          - Executable code (Python/Bash/etc.)\n" +
                "    ├── references/       - Documentation intended to be loaded into context as needed\n" +
                "    └── assets/           - Files used in output (templates, icons, fonts, etc.)\n" +
                "```\n" +
                "\n" +
                "#### SKILL.md (required)\n" +
                "\n" +
                "**Metadata Quality:** The `name` and `description` in YAML frontmatter determine when Claude will use the skill. Be specific about what the skill does and when to use it. Use the third-person (e.g. \"This skill should be used when...\" instead of \"Use this skill when...\").\n" +
                "\n" +
                "#### Bundled Resources (optional)\n" +
                "\n" +
                "##### Scripts (`scripts/`)\n" +
                "\n" +
                "Executable code (Python/Bash/etc.) for tasks that require deterministic reliability or are repeatedly rewritten.\n" +
                "\n" +
                "- **When to include**: When the same code is being rewritten repeatedly or deterministic reliability is needed\n" +
                "- **Example**: `scripts/rotate_pdf.py` for PDF rotation tasks\n" +
                "- **Benefits**: Token efficient, deterministic, may be executed without loading into context\n" +
                "- **Note**: Scripts may still need to be read by Claude for patching or environment-specific adjustments\n" +
                "\n" +
                "##### References (`references/`)\n" +
                "\n" +
                "Documentation and reference material intended to be loaded as needed into context to inform Claude's process and thinking.\n" +
                "\n" +
                "- **When to include**: For documentation that Claude should reference while working\n" +
                "- **Examples**: `references/finance.md` for financial schemas, `references/mnda.md` for company NDA template, `references/policies.md` for company policies, `references/api_docs.md` for API specifications\n" +
                "- **Use cases**: Database schemas, API documentation, domain knowledge, company policies, detailed workflow guides\n" +
                "- **Benefits**: Keeps SKILL.md lean, loaded only when Claude determines it's needed\n" +
                "- **Best practice**: If files are large (>10k words), include grep search patterns in SKILL.md\n" +
                "- **Avoid duplication**: Information should live in either SKILL.md or references files, not both. Prefer references files for detailed information unless it's truly core to the skill—this keeps SKILL.md lean while making information discoverable without hogging the context window. Keep only essential procedural instructions and workflow guidance in SKILL.md; move detailed reference material, schemas, and examples to references files.\n" +
                "\n" +
                "##### Assets (`assets/`)\n" +
                "\n" +
                "Files not intended to be loaded into context, but rather used within the output Claude produces.\n" +
                "\n" +
                "- **When to include**: When the skill needs files that will be used in the final output\n" +
                "- **Examples**: `assets/logo.png` for brand assets, `assets/slides.pptx` for PowerPoint templates, `assets/frontend-template/` for HTML/React boilerplate, `assets/font.ttf` for typography\n" +
                "- **Use cases**: Templates, images, icons, boilerplate code, fonts, sample documents that get copied or modified\n" +
                "- **Benefits**: Separates output resources from documentation, enables Claude to use files without loading them into context\n" +
                "\n" +
                "### Progressive Disclosure Design Principle\n" +
                "\n" +
                "Skills use a three-level loading system to manage context efficiently:\n" +
                "\n" +
                "1. **Metadata (name + description)** - Always in context (~100 words)\n" +
                "2. **SKILL.md body** - When skill triggers (<5k words)\n" +
                "3. **Bundled resources** - As needed by Claude (Unlimited*)\n" +
                "\n" +
                "*Unlimited because scripts can be executed without reading into context window.\n" +
                "\n" +
                "## Skill Creation Process\n" +
                "\n" +
                "To create a skill, follow the \"Skill Creation Process\" in order, skipping steps only if there is a clear reason why they are not applicable.\n" +
                "\n" +
                "### Step 1: Understanding the Skill with Concrete Examples\n" +
                "\n" +
                "Skip this step only when the skill's usage patterns are already clearly understood. It remains valuable even when working with an existing skill.\n" +
                "\n" +
                "To create an effective skill, clearly understand concrete examples of how the skill will be used. This understanding can come from either direct user examples or generated examples that are validated with user feedback.\n" +
                "\n" +
                "For example, when building an image-editor skill, relevant questions include:\n" +
                "\n" +
                "- \"What functionality should the image-editor skill support? Editing, rotating, anything else?\"\n" +
                "- \"Can you give some examples of how this skill would be used?\"\n" +
                "- \"I can imagine users asking for things like 'Remove the red-eye from this image' or 'Rotate this image'. Are there other ways you imagine this skill being used?\"\n" +
                "- \"What would a user say that should trigger this skill?\"\n" +
                "\n" +
                "To avoid overwhelming users, avoid asking too many questions in a single message. Start with the most important questions and follow up as needed for better effectiveness.\n" +
                "\n" +
                "Conclude this step when there is a clear sense of the functionality the skill should support.\n" +
                "\n" +
                "### Step 2: Planning the Reusable Skill Contents\n" +
                "\n" +
                "To turn concrete examples into an effective skill, analyze each example by:\n" +
                "\n" +
                "1. Considering how to execute on the example from scratch\n" +
                "2. Identifying what scripts, references, and assets would be helpful when executing these workflows repeatedly\n" +
                "\n" +
                "Example: When building a `pdf-editor` skill to handle queries like \"Help me rotate this PDF,\" the analysis shows:\n" +
                "\n" +
                "1. Rotating a PDF requires re-writing the same code each time\n" +
                "2. A `scripts/rotate_pdf.py` script would be helpful to store in the skill\n" +
                "\n" +
                "Example: When designing a `frontend-webapp-builder` skill for queries like \"Build me a todo app\" or \"Build me a dashboard to track my steps,\" the analysis shows:\n" +
                "\n" +
                "1. Writing a frontend webapp requires the same boilerplate HTML/React each time\n" +
                "2. An `assets/hello-world/` template containing the boilerplate HTML/React project files would be helpful to store in the skill\n" +
                "\n" +
                "Example: When building a `big-query` skill to handle queries like \"How many users have logged in today?\" the analysis shows:\n" +
                "\n" +
                "1. Querying BigQuery requires re-discovering the table schemas and relationships each time\n" +
                "2. A `references/schema.md` file documenting the table schemas would be helpful to store in the skill\n" +
                "\n" +
                "To establish the skill's contents, analyze each concrete example to create a list of the reusable resources to include: scripts, references, and assets.\n" +
                "\n" +
                "### Step 3: Initializing the Skill\n" +
                "\n" +
                "At this point, it is time to actually create the skill.\n" +
                "\n" +
                "Skip this step only if the skill being developed already exists, and iteration or packaging is needed. In this case, continue to the next step.\n" +
                "\n" +
                "When creating a new skill from scratch, always run the `init_skill.py` script. The script conveniently generates a new template skill directory that automatically includes everything a skill requires, making the skill creation process much more efficient and reliable.\n" +
                "\n" +
                "Usage:\n" +
                "\n" +
                "```bash\n" +
                "scripts/init_skill.py <skill-name> --path <output-directory>\n" +
                "```\n" +
                "\n" +
                "The script:\n" +
                "\n" +
                "- Creates the skill directory at the specified path\n" +
                "- Generates a SKILL.md template with proper frontmatter and TODO placeholders\n" +
                "- Creates example resource directories: `scripts/`, `references/`, and `assets/`\n" +
                "- Adds example files in each directory that can be customized or deleted\n" +
                "\n" +
                "After initialization, customize or remove the generated SKILL.md and example files as needed.\n" +
                "\n" +
                "### Step 4: Edit the Skill\n" +
                "\n" +
                "When editing the (newly-generated or existing) skill, remember that the skill is being created for another instance of Claude to use. Focus on including information that would be beneficial and non-obvious to Claude. Consider what procedural knowledge, domain-specific details, or reusable assets would help another Claude instance execute these tasks more effectively.\n" +
                "\n" +
                "#### Start with Reusable Skill Contents\n" +
                "\n" +
                "To begin implementation, start with the reusable resources identified above: `scripts/`, `references/`, and `assets/` files. Note that this step may require user input. For example, when implementing a `brand-guidelines` skill, the user may need to provide brand assets or templates to store in `assets/`, or documentation to store in `references/`.\n" +
                "\n" +
                "Also, delete any example files and directories not needed for the skill. The initialization script creates example files in `scripts/`, `references/`, and `assets/` to demonstrate structure, but most skills won't need all of them.\n" +
                "\n" +
                "#### Update SKILL.md\n" +
                "\n" +
                "**Writing Style:** Write the entire skill using **imperative/infinitive form** (verb-first instructions), not second person. Use objective, instructional language (e.g., \"To accomplish X, do Y\" rather than \"You should do X\" or \"If you need to do X\"). This maintains consistency and clarity for AI consumption.\n" +
                "\n" +
                "To complete SKILL.md, answer the following questions:\n" +
                "\n" +
                "1. What is the purpose of the skill, in a few sentences?\n" +
                "2. When should the skill be used?\n" +
                "3. In practice, how should Claude use the skill? All reusable skill contents developed above should be referenced so that Claude knows how to use them.\n" +
                "\n" +
                "### Step 5: Packaging a Skill\n" +
                "\n" +
                "Once the skill is ready, it should be packaged into a distributable zip file that gets shared with the user. The packaging process automatically validates the skill first to ensure it meets all requirements:\n" +
                "\n" +
                "```bash\n" +
                "scripts/package_skill.py <path/to/skill-folder>\n" +
                "```\n" +
                "\n" +
                "Optional output directory specification:\n" +
                "\n" +
                "```bash\n" +
                "scripts/package_skill.py <path/to/skill-folder> ./dist\n" +
                "```\n" +
                "\n" +
                "The packaging script will:\n" +
                "\n" +
                "1. **Validate** the skill automatically, checking:\n" +
                "   - YAML frontmatter format and required fields\n" +
                "   - Skill naming conventions and directory structure\n" +
                "   - Description completeness and quality\n" +
                "   - File organization and resource references\n" +
                "\n" +
                "2. **Package** the skill if validation passes, creating a zip file named after the skill (e.g., `my-skill.zip`) that includes all files and maintains the proper directory structure for distribution.\n" +
                "\n" +
                "If validation fails, the script will report the errors and exit without creating a package. Fix any validation errors and run the packaging command again.\n" +
                "\n" +
                "### Step 6: Iterate\n" +
                "\n" +
                "After testing the skill, users may request improvements. Often this happens right after using the skill, with fresh context of how the skill performed.\n" +
                "\n" +
                "**Iteration workflow:**\n" +
                "1. Use the skill on real tasks\n" +
                "2. Notice struggles or inefficiencies\n" +
                "3. Identify how SKILL.md or bundled resources should be updated\n" +
                "4. Implement changes and test again\n" +
                "The above is the full content of resource `.md` in skill `skill-creator`\n", content);
    }

    @Test
    public void testFetchOverlap5() throws Exception {
        FileSystemFetcher fileSystemFetcher = new FileSystemFetcher();
        fileSystemFetcher.setDir(System.getProperty("user.dir") + "/src/test/resources/skills");
        fileSystemFetcher.setPlaceholderResolver(ObjectBuilder.buildEmptyPlaceholderResolver());
        fileSystemFetcher.setResourceService(ObjectBuilder.buildResourceService());
        fileSystemFetcher.setUsage("USAGE");
        fileSystemFetcher.setExpire(100000);
        fileSystemFetcher.setCached(true);
        fileSystemFetcher.init();
        fileSystemFetcher.fetchSkills(ObjectBuilder.buildWorkflowTask());
        String content = fileSystemFetcher.fetchResource(ObjectBuilder.buildWorkflowTask(), "skill-creator", "");
        assertEquals("\n" +
                "The following is the beginning of resource `` in skill `skill-creator`\n" +
                "# Skill Creator\n" +
                "\n" +
                "This skill provides guidance for creating effective skills.\n" +
                "\n" +
                "## About Skills\n" +
                "\n" +
                "Skills are modular, self-contained packages that extend Claude's capabilities by providing\n" +
                "specialized knowledge, workflows, and tools. Think of them as \"onboarding guides\" for specific\n" +
                "domains or tasks—they transform Claude from a general-purpose agent into a specialized agent\n" +
                "equipped with procedural knowledge that no model can fully possess.\n" +
                "\n" +
                "### What Skills Provide\n" +
                "\n" +
                "1. Specialized workflows - Multi-step procedures for specific domains\n" +
                "2. Tool integrations - Instructions for working with specific file formats or APIs\n" +
                "3. Domain expertise - Company-specific knowledge, schemas, business logic\n" +
                "4. Bundled resources - Scripts, references, and assets for complex and repetitive tasks\n" +
                "\n" +
                "### Anatomy of a Skill\n" +
                "\n" +
                "Every skill consists of a required SKILL.md file and optional bundled resources:\n" +
                "\n" +
                "```\n" +
                "skill-name/\n" +
                "├── SKILL.md (required)\n" +
                "│   ├── YAML frontmatter metadata (required)\n" +
                "│   │   ├── name: (required)\n" +
                "│   │   └── description: (required)\n" +
                "│   └── Markdown instructions (required)\n" +
                "└── Bundled Resources (optional)\n" +
                "    ├── scripts/          - Executable code (Python/Bash/etc.)\n" +
                "    ├── references/       - Documentation intended to be loaded into context as needed\n" +
                "    └── assets/           - Files used in output (templates, icons, fonts, etc.)\n" +
                "```\n" +
                "\n" +
                "#### SKILL.md (required)\n" +
                "\n" +
                "**Metadata Quality:** The `name` and `description` in YAML frontmatter determine when Claude will use the skill. Be specific about what the skill does and when to use it. Use the third-person (e.g. \"This skill should be used when...\" instead of \"Use this skill when...\").\n" +
                "\n" +
                "#### Bundled Resources (optional)\n" +
                "\n" +
                "##### Scripts (`scripts/`)\n" +
                "\n" +
                "Executable code (Python/Bash/etc.) for tasks that require deterministic reliability or are repeatedly rewritten.\n" +
                "\n" +
                "- **When to include**: When the same code is being rewritten repeatedly or deterministic reliability is needed\n" +
                "- **Example**: `scripts/rotate_pdf.py` for PDF rotation tasks\n" +
                "- **Benefits**: Token efficient, deterministic, may be executed without loading into context\n" +
                "- **Note**: Scripts may still need to be read by Claude for patching or environment-specific adjustments\n" +
                "\n" +
                "##### References (`references/`)\n" +
                "\n" +
                "Documentation and reference material intended to be loaded as needed into context to inform Claude's process and thinking.\n" +
                "\n" +
                "- **When to include**: For documentation that Claude should reference while working\n" +
                "- **Examples**: `references/finance.md` for financial schemas, `references/mnda.md` for company NDA template, `references/policies.md` for company policies, `references/api_docs.md` for API specifications\n" +
                "- **Use cases**: Database schemas, API documentation, domain knowledge, company policies, detailed workflow guides\n" +
                "- **Benefits**: Keeps SKILL.md lean, loaded only when Claude determines it's needed\n" +
                "- **Best practice**: If files are large (>10k words), include grep search patterns in SKILL.md\n" +
                "- **Avoid duplication**: Information should live in either SKILL.md or references files, not both. Prefer references files for detailed information unless it's truly core to the skill—this keeps SKILL.md lean while making information discoverable without hogging the context window. Keep only essential procedural instructions and workflow guidance in SKILL.md; move detailed reference material, schemas, and examples to references files.\n" +
                "\n" +
                "##### Assets (`assets/`)\n" +
                "\n" +
                "Files not intended to be loaded into context, but rather used within the output Claude produces.\n" +
                "\n" +
                "- **When to include**: When the skill needs files that will be used in the final output\n" +
                "- **Examples**: `assets/logo.png` for brand assets, `assets/slides.pptx` for PowerPoint templates, `assets/frontend-template/` for HTML/React boilerplate, `assets/font.ttf` for typography\n" +
                "- **Use cases**: Templates, images, icons, boilerplate code, fonts, sample documents that get copied or modified\n" +
                "- **Benefits**: Separates output resources from documentation, enables Claude to use files without loading them into context\n" +
                "\n" +
                "### Progressive Disclosure Design Principle\n" +
                "\n" +
                "Skills use a three-level loading system to manage context efficiently:\n" +
                "\n" +
                "1. **Metadata (name + description)** - Always in context (~100 words)\n" +
                "2. **SKILL.md body** - When skill triggers (<5k words)\n" +
                "3. **Bundled resources** - As needed by Claude (Unlimited*)\n" +
                "\n" +
                "*Unlimited because scripts can be executed without reading into context window.\n" +
                "\n" +
                "## Skill Creation Process\n" +
                "\n" +
                "To create a skill, follow the \"Skill Creation Process\" in order, skipping steps only if there is a clear reason why they are not applicable.\n" +
                "\n" +
                "### Step 1: Understanding the Skill with Concrete Examples\n" +
                "\n" +
                "Skip this step only when the skill's usage patterns are already clearly understood. It remains valuable even when working with an existing skill.\n" +
                "\n" +
                "To create an effective skill, clearly understand concrete examples of how the skill will be used. This understanding can come from either direct user examples or generated examples that are validated with user feedback.\n" +
                "\n" +
                "For example, when building an image-editor skill, relevant questions include:\n" +
                "\n" +
                "- \"What functionality should the image-editor skill support? Editing, rotating, anything else?\"\n" +
                "- \"Can you give some examples of how this skill would be used?\"\n" +
                "- \"I can imagine users asking for things like 'Remove the red-eye from this image' or 'Rotate this image'. Are there other ways you imagine this skill being used?\"\n" +
                "- \"What would a user say that should trigger this skill?\"\n" +
                "\n" +
                "To avoid overwhelming users, avoid asking too many questions in a single message. Start with the most important questions and follow up as needed for better effectiveness.\n" +
                "\n" +
                "Conclude this step when there is a clear sense of the functionality the skill should support.\n" +
                "\n" +
                "### Step 2: Planning the Reusable Skill Contents\n" +
                "\n" +
                "To turn concrete examples into an effective skill, analyze each example by:\n" +
                "\n" +
                "1. Considering how to execute on the example from scratch\n" +
                "2. Identifying what scripts, references, and assets would be helpful when executing these workflows repeatedly\n" +
                "\n" +
                "Example: When building a `pdf-editor` skill to handle queries like \"Help me rotate this PDF,\" the analysis shows:\n" +
                "\n" +
                "1. Rotating a PDF requires re-writing the same code each time\n" +
                "2. A `scripts/rotate_pdf.py` script would be helpful to store in the skill\n" +
                "\n" +
                "Example: When designing a `frontend-webapp-builder` skill for queries like \"Build me a todo app\" or \"Build me a dashboard to track my steps,\" the analysis shows:\n" +
                "\n" +
                "1. Writing a frontend webapp requires the same boilerplate HTML/React each time\n" +
                "2. An `assets/hello-world/` template containing the boilerplate HTML/React project files would be helpful to store in the skill\n" +
                "\n" +
                "Example: When building a `big-query` skill to handle queries like \"How many users have logged in today?\" the analysis shows:\n" +
                "\n" +
                "1. Querying BigQuery requires re-discovering the table schemas and relationships each time\n" +
                "2. A `references/schema.md` file documenting the table schemas would be helpful to store in the skill\n" +
                "\n" +
                "To establish the skill's contents, analyze each concrete example to create a list of the reusable resources to include: scripts, references, and assets.\n" +
                "\n" +
                "### Step 3: Initializing the Skill\n" +
                "\n" +
                "At this point, it is time to actually create the skill.\n" +
                "\n" +
                "Skip this step only if the skill being developed already exists, and iteration or packaging is needed. In this case, continue to the next step.\n" +
                "\n" +
                "When creating a new skill from scratch, always run the `init_skill.py` script. The script conveniently generates a new template skill directory that automatically includes everything a skill requires, making the skill creation process much more efficient and reliable.\n" +
                "\n" +
                "Usage:\n" +
                "\n" +
                "```bash\n" +
                "scripts/init_skill.py <skill-name> --path <output-directory>\n" +
                "```\n" +
                "\n" +
                "The script:\n" +
                "\n" +
                "- Creates the skill directory at the specified path\n" +
                "- Generates a SKILL.md template with proper frontmatter and TODO placeholders\n" +
                "- Creates example resource directories: `scripts/`, `references/`, and `assets/`\n" +
                "- Adds example files in each directory that can be customized or deleted\n" +
                "\n" +
                "After initialization, customize or remove the generated SKILL.md and example files as needed.\n" +
                "\n" +
                "### Step 4: Edit the Skill\n" +
                "\n" +
                "When editing the (newly-generated or existing) skill, remember that the skill is being created for another instance of Claude to use. Focus on including information that would be beneficial and non-obvious to Claude. Consider what procedural knowledge, domain-specific details, or reusable assets would help another Claude instance execute these tasks more effectively.\n" +
                "\n" +
                "#### Start with Reusable Skill Contents\n" +
                "\n" +
                "To begin implementation, start with the reusable resources identified above: `scripts/`, `references/`, and `assets/` files. Note that this step may require user input. For example, when implementing a `brand-guidelines` skill, the user may need to provide brand assets or templates to store in `assets/`, or documentation to store in `references/`.\n" +
                "\n" +
                "Also, delete any example files and directories not needed for the skill. The initialization script creates example files in `scripts/`, `references/`, and `assets/` to demonstrate structure, but most skills won't need all of them.\n" +
                "\n" +
                "#### Update SKILL.md\n" +
                "\n" +
                "**Writing Style:** Write the entire skill using **imperative/infinitive form** (verb-first instructions), not second person. Use objective, instructional language (e.g., \"To accomplish X, do Y\" rather than \"You should do X\" or \"If you need to do X\"). This maintains consistency and clarity for AI consumption.\n" +
                "\n" +
                "To complete SKILL.md, answer the following questions:\n" +
                "\n" +
                "1. What is the purpose of the skill, in a few sentences?\n" +
                "2. When should the skill be used?\n" +
                "3. In practice, how should Claude use the skill? All reusable skill contents developed above should be referenced so that Claude knows how to use them.\n" +
                "\n" +
                "### Step 5: Packaging a Skill\n" +
                "\n" +
                "Once the skill is ready, it should be packaged into a distributable zip file that gets shared with the user. The packaging process automatically validates the skill first to ensure it meets all requirements:\n" +
                "\n" +
                "```bash\n" +
                "scripts/package_skill.py <path/to/skill-folder>\n" +
                "```\n" +
                "\n" +
                "Optional output directory specification:\n" +
                "\n" +
                "```bash\n" +
                "scripts/package_skill.py <path/to/skill-folder> ./dist\n" +
                "```\n" +
                "\n" +
                "The packaging script will:\n" +
                "\n" +
                "1. **Validate** the skill automatically, checking:\n" +
                "   - YAML frontmatter format and required fields\n" +
                "   - Skill naming conventions and directory structure\n" +
                "   - Description completeness and quality\n" +
                "   - File organization and resource references\n" +
                "\n" +
                "2. **Package** the skill if validation passes, creating a zip file named after the skill (e.g., `my-skill.zip`) that includes all files and maintains the proper directory structure for distribution.\n" +
                "\n" +
                "If validation fails, the script will report the errors and exit without creating a package. Fix any validation errors and run the packaging command again.\n" +
                "\n" +
                "### Step 6: Iterate\n" +
                "\n" +
                "After testing the skill, users may request improvements. Often this happens right after using the skill, with fresh context of how the skill performed.\n" +
                "\n" +
                "**Iteration workflow:**\n" +
                "1. Use the skill on real tasks\n" +
                "2. Notice struggles or inefficiencies\n" +
                "3. Identify how SKILL.md or bundled resources should be updated\n" +
                "4. Implement changes and test again\n" +
                "The above is the full content of resource `` in skill `skill-creator`\n", content);
    }

    @Test
    public void testBasicPath() throws Exception {
        // 1. 局部实例化，无全局变量
        FileSystemFetcher service = new FileSystemFetcher();
        service.name = "skills";
        // 假设该字段可访问或通过构造函数设置
        // 分支：普通路径，不命中任何 IF
        String result = service.normalizePath("java", "docs/readme.md");
        assertEquals("docs/readme.md", result);
    }

    @Test
    public void testSeparatorAndSubstring() throws Exception {
        FileSystemFetcher service = new FileSystemFetcher();
        service.name = "skills";
        // 分支：覆盖以分隔符开头的情况
        // 注意：原逻辑 substring(1, len-1) 会导致 "/a.md" 变成 "a.m"
        String pathWithSep = File.separator + "a.md";
        String result = service.normalizePath("java", pathWithSep);
        assertEquals("a.md", result);
    }

    @Test
    public void testRemoveThisNamePrefix() throws Exception {
        FileSystemFetcher service = new FileSystemFetcher();
        service.name = "skills";
        // 分支：命中第一个 IF (this.name + File.separator)
        String input = "skills" + File.separator + "test.md";
        String result = service.normalizePath("java", input);
        assertEquals("test.md", result);
    }

    @Test
    public void testRemoveParamNamePrefix() throws Exception {
        FileSystemFetcher service = new FileSystemFetcher();
        service.name = "skills";
        // 分支：命中第二个 IF (name + File.separator)
        String input = "java" + File.separator + "install.md";
        String result = service.normalizePath("java", input);
        assertEquals("install.md", result);
    }

    @Test
    public void testEmptyPathToDefault() throws Exception {
        FileSystemFetcher service = new FileSystemFetcher();
        service.name = "skills";
        // 分支：StringUtils.isEmpty(path)
        String result = service.normalizePath("java", "");
        assertEquals("SKILL.md", result);
    }

    @Test
    public void testMdExtensionToDefault() throws Exception {
        FileSystemFetcher service = new FileSystemFetcher();
        service.name = "skills";
        // 分支：StringUtils.equalsIgnoreCase(path, ".md")
        String result = service.normalizePath("java", ".md");
        assertEquals("SKILL.md", result);
    }

    @Test
    public void testDoublePrefixRemoval() throws Exception {
        FileSystemFetcher service = new FileSystemFetcher();
        service.name = "skills";
        // 分支：连续命中两个移除前缀的 IF
        String input = "skills" + File.separator + "java" + File.separator + "core.md";
        String result = service.normalizePath("java", input);
        assertEquals("java/core.md", result);
    }

    @Test
    public void testIsBinaryWithExtensionPng() throws Exception {
        FileSystemFetcher fetcher = new FileSystemFetcher();
        fetcher.setPlaceholderResolver(ObjectBuilder.buildEmptyPlaceholderResolver());
        Assert.assertTrue(fetcher.isBinary(ObjectBuilder.buildWorkflowTask(), "skill", "image.png"));
    }

    @Test
    public void testIsBinaryWithExtensionTxt() throws Exception {
        FileSystemFetcher fetcher = new FileSystemFetcher();
        fetcher.setPlaceholderResolver(ObjectBuilder.buildEmptyPlaceholderResolver());
        Assert.assertFalse(fetcher.isBinary(ObjectBuilder.buildWorkflowTask(), "skill", "readme.txt"));
    }

    @Test
    public void testIsBinaryWithPathContainingSeparator() throws Exception {
        FileSystemFetcher fetcher = new FileSystemFetcher();
        fetcher.setPlaceholderResolver(ObjectBuilder.buildEmptyPlaceholderResolver());
        Assert.assertTrue(fetcher.isBinary(ObjectBuilder.buildWorkflowTask(), "skill", "sub/dir/file.pdf"));
    }

    @Test
    public void testIsBinaryWithNoExtension() throws Exception {
        FileSystemFetcher fetcher = new FileSystemFetcher();
        fetcher.setPlaceholderResolver(ObjectBuilder.buildEmptyPlaceholderResolver());
        Assert.assertTrue(fetcher.isBinary(ObjectBuilder.buildWorkflowTask(), "skill", "Makefile"));
    }

    @Test
    public void testIsBinaryWithEmptyPath() throws Exception {
        FileSystemFetcher fetcher = new FileSystemFetcher();
        fetcher.setPlaceholderResolver(ObjectBuilder.buildEmptyPlaceholderResolver());
        Assert.assertTrue(fetcher.isBinary(ObjectBuilder.buildWorkflowTask(), "skill", ""));
    }

    @Test
    public void testBuildContentWhenNotBinaryReturnsPrefixReplaceAndSuffix() throws Exception {
        FileSystemFetcher fetcher = new FileSystemFetcher() {
            @Override
            protected FileSystemFetcher.SkillFetchCallable buildSkillFetchCallable(Boolean tolerated, Boolean binary, String path) {
                return new FileSystemFetcher.SkillFetchCallable(
                        ObjectBuilder.buildEmptyPlaceholderResolver(),
                        ObjectBuilder.buildResourceService(),
                        tolerated,
                        binary,
                        path) {
                    @Override
                    public String call() throws Exception {
                        return "Line1";
                    }
                };
            }
        };
        fetcher.setPlaceholderResolver(ObjectBuilder.buildEmptyPlaceholderResolver());
        fetcher.setPrefix("#");
        WorkflowTask task = ObjectBuilder.buildWorkflowTask();
        String name = "mySkill";
        String path = "res.txt";
        String location = "/ignored";
        String nl = System.lineSeparator();
        String expected = nl + "The following is the beginning of resource `" + path + "` in skill `" + name + "`" + nl
                + "Line1"
                + nl + "The above is the full content of resource `" + path + "` in skill `" + name + "`" + nl;
        assertEquals(expected, fetcher.buildContent(task, location, false, name, path));
    }

    @Test
    public void testBuildContentWhenBinaryReturnsRawResourceOnly() throws Exception {
        final String rawPayload = "H4sIAAAAAAAA";
        FileSystemFetcher fetcher = new FileSystemFetcher() {
            @Override
            protected FileSystemFetcher.SkillFetchCallable buildSkillFetchCallable(Boolean tolerated, Boolean binary, String path) {
                return new FileSystemFetcher.SkillFetchCallable(
                        ObjectBuilder.buildEmptyPlaceholderResolver(),
                        ObjectBuilder.buildResourceService(),
                        tolerated,
                        binary,
                        path) {
                    @Override
                    public String call() throws Exception {
                        return rawPayload;
                    }
                };
            }
        };
        fetcher.setPlaceholderResolver(ObjectBuilder.buildEmptyPlaceholderResolver());
        fetcher.setPrefix("#");
        WorkflowTask task = ObjectBuilder.buildWorkflowTask();
        assertEquals(rawPayload, fetcher.buildContent(task, "/any", true, "n", "f.png"));
    }
}
