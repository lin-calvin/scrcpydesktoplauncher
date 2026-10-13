package com.calvin.alauncher;

import android.content.Context;
import android.content.Intent;
import android.content.pm.ApplicationInfo;
import android.content.pm.PackageManager;
import android.graphics.Bitmap;
import android.graphics.Canvas;
import android.graphics.drawable.Drawable;

import java.io.File;
import java.io.FileOutputStream;
import java.io.FileWriter;
import java.io.PrintWriter;
import java.util.List;

// Runs on-device via app_process (no APK / no root) to dump the launchable
// apps and their rendered launcher icons:
//   app_process /system/bin com.calvin.alauncher.AppList <outDir>
public class AppList {
    static final int SIZE = 192;

    public static void main(String[] args) throws Exception {
        String outDir = args.length > 0 ? args[0] : "/data/local/tmp/scrcpydesktoplauncher";
        Context ctx = getContext();
        PackageManager pm = ctx.getPackageManager();

        File base = new File(outDir);
        File icons = new File(base, "icons");
        icons.mkdirs();

        List<ApplicationInfo> apps = pm.getInstalledApplications(0);
        PrintWriter mf = new PrintWriter(new FileWriter(new File(base, "manifest.jsonl")));

        int n = 0;
        for (ApplicationInfo ai : apps) {
            Intent li;
            try {
                li = pm.getLaunchIntentForPackage(ai.packageName);
            } catch (Throwable t) {
                li = null;
            }
            if (li == null) continue; // no launcher entry -> skip

            String label;
            try {
                label = String.valueOf(pm.getApplicationLabel(ai));
            } catch (Throwable t) {
                label = ai.packageName;
            }
            boolean system = (ai.flags & ApplicationInfo.FLAG_SYSTEM) != 0;

            Drawable d;
            try {
                d = li.getComponent() != null
                        ? pm.getActivityIcon(li.getComponent())
                        : pm.getApplicationIcon(ai.packageName);
            } catch (Throwable t) {
                try {
                    d = pm.getApplicationIcon(ai.packageName);
                } catch (Throwable t2) {
                    continue;
                }
            }

            try {
                Bitmap bmp = Bitmap.createBitmap(SIZE, SIZE, Bitmap.Config.ARGB_8888);
                Canvas c = new Canvas(bmp);
                d.setBounds(0, 0, SIZE, SIZE);
                d.draw(c);
                FileOutputStream o = new FileOutputStream(new File(icons, ai.packageName + ".png"));
                bmp.compress(Bitmap.CompressFormat.PNG, 100, o);
                o.close();
                bmp.recycle();
            } catch (Throwable t) {
                continue;
            }

            mf.println(line(ai.packageName, label, system));
            n++;
        }
        mf.flush();
        mf.close();
        System.out.println("apps=" + n + " out=" + base.getAbsolutePath());
        System.exit(0);
    }

    static String line(String pkg, String label, boolean system) {
        return "{\"package\":" + q(pkg)
                + ",\"label\":" + q(label)
                + ",\"system\":" + system
                + ",\"icon\":" + q(pkg + ".png") + "}";
    }

    static String q(String s) {
        StringBuilder b = new StringBuilder("\"");
        for (int i = 0; i < s.length(); i++) {
            char ch = s.charAt(i);
            switch (ch) {
                case '"': b.append("\\\""); break;
                case '\\': b.append("\\\\"); break;
                case '\n': b.append("\\n"); break;
                case '\r': b.append("\\r"); break;
                case '\t': b.append("\\t"); break;
                default:
                    if (ch < 0x20) b.append(String.format("\\u%04x", (int) ch));
                    else b.append(ch);
            }
        }
        return b.append("\"").toString();
    }

    static Context getContext() throws Exception {
        android.os.Looper.prepare();
        Class<?> at = Class.forName("android.app.ActivityThread");
        Object thread = at.getMethod("systemMain").invoke(null);
        return (Context) at.getMethod("getSystemContext").invoke(thread);
    }
}
