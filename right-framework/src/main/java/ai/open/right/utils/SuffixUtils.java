
package ai.open.right.utils;

import lombok.extern.slf4j.Slf4j;
import org.apache.commons.lang3.StringUtils;

import java.nio.file.Files;
import java.nio.file.Paths;
import java.util.HashSet;
import java.util.Set;

@Slf4j
public class SuffixUtils {

    public static final Set<String> MIME_TYPE = new HashSet<String>();

    static {
        try {
            // Files.probeContentType平台相关，可能为Null
            SuffixUtils.MIME_TYPE.add(Files.probeContentType(Paths.get("t.torrent")));
            SuffixUtils.MIME_TYPE.add(Files.probeContentType(Paths.get("t.sqlite")));
            SuffixUtils.MIME_TYPE.add(Files.probeContentType(Paths.get("t.class")));
            SuffixUtils.MIME_TYPE.add(Files.probeContentType(Paths.get("t.pptx")));
            SuffixUtils.MIME_TYPE.add(Files.probeContentType(Paths.get("t.docx")));
            SuffixUtils.MIME_TYPE.add(Files.probeContentType(Paths.get("t.xlsx")));
            SuffixUtils.MIME_TYPE.add(Files.probeContentType(Paths.get("t.flac")));
            SuffixUtils.MIME_TYPE.add(Files.probeContentType(Paths.get("t.jpeg")));
            SuffixUtils.MIME_TYPE.add(Files.probeContentType(Paths.get("t.font")));
            SuffixUtils.MIME_TYPE.add(Files.probeContentType(Paths.get("t.psd")));
            SuffixUtils.MIME_TYPE.add(Files.probeContentType(Paths.get("t.png")));
            SuffixUtils.MIME_TYPE.add(Files.probeContentType(Paths.get("t.zip")));
            SuffixUtils.MIME_TYPE.add(Files.probeContentType(Paths.get("t.jpg")));
            SuffixUtils.MIME_TYPE.add(Files.probeContentType(Paths.get("t.pdf")));
            SuffixUtils.MIME_TYPE.add(Files.probeContentType(Paths.get("t.ppt")));
            SuffixUtils.MIME_TYPE.add(Files.probeContentType(Paths.get("t.exe")));
            SuffixUtils.MIME_TYPE.add(Files.probeContentType(Paths.get("t.dll")));
            SuffixUtils.MIME_TYPE.add(Files.probeContentType(Paths.get("t.bin")));
            SuffixUtils.MIME_TYPE.add(Files.probeContentType(Paths.get("t.bmp")));
            SuffixUtils.MIME_TYPE.add(Files.probeContentType(Paths.get("t.mp3")));
            SuffixUtils.MIME_TYPE.add(Files.probeContentType(Paths.get("t.mp4")));
            SuffixUtils.MIME_TYPE.add(Files.probeContentType(Paths.get("t.mov")));
            SuffixUtils.MIME_TYPE.add(Files.probeContentType(Paths.get("t.avi")));
            SuffixUtils.MIME_TYPE.add(Files.probeContentType(Paths.get("t.wav")));
            SuffixUtils.MIME_TYPE.add(Files.probeContentType(Paths.get("t.doc")));
            SuffixUtils.MIME_TYPE.add(Files.probeContentType(Paths.get("t.xls")));
            SuffixUtils.MIME_TYPE.add(Files.probeContentType(Paths.get("t.rar")));
            SuffixUtils.MIME_TYPE.add(Files.probeContentType(Paths.get("t.tar")));
            SuffixUtils.MIME_TYPE.add(Files.probeContentType(Paths.get("t.bz2")));
            SuffixUtils.MIME_TYPE.add(Files.probeContentType(Paths.get("t.pyc")));
            SuffixUtils.MIME_TYPE.add(Files.probeContentType(Paths.get("t.dat")));
            SuffixUtils.MIME_TYPE.add(Files.probeContentType(Paths.get("t.iso")));
            SuffixUtils.MIME_TYPE.add(Files.probeContentType(Paths.get("t.ttf")));
            SuffixUtils.MIME_TYPE.add(Files.probeContentType(Paths.get("t.otf")));
            SuffixUtils.MIME_TYPE.add(Files.probeContentType(Paths.get("t.swf")));
            SuffixUtils.MIME_TYPE.add(Files.probeContentType(Paths.get("t.elf")));
            SuffixUtils.MIME_TYPE.add(Files.probeContentType(Paths.get("t.obj")));
            SuffixUtils.MIME_TYPE.add(Files.probeContentType(Paths.get("t.7z")));
            SuffixUtils.MIME_TYPE.add(Files.probeContentType(Paths.get("t.gz")));
            SuffixUtils.MIME_TYPE.add(Files.probeContentType(Paths.get("t.db")));
            SuffixUtils.MIME_TYPE.add(Files.probeContentType(Paths.get("t.so")));
            SuffixUtils.MIME_TYPE.add(Files.probeContentType(Paths.get("t.o")));
        } catch (Exception e) {
            if (log.isWarnEnabled()) {
                log.warn(e.getMessage(), e);
            }
        }
    }

    // 当前后缀名是否为二进制格式
    public static Boolean isBinary(String suffix) throws Exception {
        if (StringUtils.isEmpty(suffix)) {
            return true;
        }
        // 空默认为二进制
        String contentType = Files.probeContentType(Paths.get("t." + suffix));
        return !StringUtils.isEmpty(contentType) && SuffixUtils.MIME_TYPE.contains(contentType);
    }
}
