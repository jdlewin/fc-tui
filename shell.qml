// Quickshell frontend - runs on your Linux box, not on Mac.
// Reads feed.json written by backend/. Keeps memory low: 15 rows, text only, no images.
// Copy this folder to Linux and run: quickshell -c .
import QtQuick
import QtQuick.Layouts
import Quickshell
import Quickshell.Io

ShellRoot {
  PanelWindow {
    anchors {
      top: true
      left: true
      right: true
    }
    implicitHeight: 420
    color: "#111111"

    ColumnLayout {
      anchors.fill: parent
      anchors.margins: 12
      spacing: 8

      Text {
        text: "Farcaster — following (fid " + (feedFile.fid ?? "3") + ")"
        color: "white"
        font.pixelSize: 16
        font.bold: true
      }

      ListView {
        Layout.fillWidth: true
        Layout.fillHeight: true
        model: ScriptModel {
          values: feedAdapter.casts ?? []
        }
        spacing: 6
        delegate: Rectangle {
          width: ListView.view.width
          height: col.height + 12
          color: "#1c1c1c"
          radius: 8
          Column {
            id: col
            anchors.left: parent.left
            anchors.right: parent.right
            anchors.margins: 8
            Text {
              width: parent.width
              text: "@" + (modelData.author?.username ?? "unknown")
              color: "#8b5cf6"
              font.pixelSize: 13
              font.bold: true
              elide: Text.ElideRight
            }
            Text {
              width: parent.width
              text: modelData.text ?? ""
              color: "#e5e5e5"
              font.pixelSize: 13
              wrapMode: Text.Wrap
              maximumLineCount: 4
              elide: Text.ElideRight
            }
          }
        }
      }

      Text {
        text: "Refresh: `go run .` in backend/ — file watches feed.json"
        color: "#666666"
        font.pixelSize: 11
      }
    }
  }

  // Watches feed.json produced by Go backend
  FileView {
    id: feedFile
    path: Qt.resolvedUrl("./feed.json")
    watchChanges: true
    onFileChanged: feedAdapter.reload()
    property string fid: "3"
  }

  JsonAdapter {
    id: feedAdapter
    json: feedFile.text() != "" ? JSON.parse(feedFile.text()) : ({casts: []})
    property var casts: (json.casts ?? []).slice(0, 15)
  }
}
