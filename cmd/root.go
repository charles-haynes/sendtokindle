// Copyright © 2017 Charles Haynes
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
// THE SOFTWARE.

package cmd

import (
	"encoding/base64"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net"
	"net/smtp"
	"os"
	"path"
	"strings"
	"time"

	homedir "github.com/mitchellh/go-homedir"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var cfgFile string

// RootCmd represents the base command when called without any subcommands
var RootCmd = &cobra.Command{
	Use:   "sendtokindle",
	Short: "Sends a mobi file to your kindle",
	Args:  cobra.ExactArgs(2),
	Long: `Given a kindle address and a mobi file, encodes that file and sends it as
an email message to the kindle server.

Example:

sendtokindle ds8vKv8V7fkM somefile.mobi`,
	// Uncomment the following line if your bare application
	// has an action associated with it:
	Run: sendToKindle,
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := RootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

const msgFmt = "MIME-Version: 1.0\r\n" +
	"Message-ID: <CAMgUg7badnLexYVc475ejJzeK97vdok9dKw23pisL6uWgur-Qw@mail.gmail.com>\r\n" +
	"Date: Wed, 22 Nov 2017 10:21:29 +0000\r\n" +
	"Subject: For kindle\r\n" +
	"From: Charles Haynes <haynes@edgeplay.org>\r\n" +
	"To: %s\r\n" +
	"Content-Type: multipart/mixed; boundary=\"f403043895ccc776c6055e8fae42\"\r\n" +
	"\r\n" +
	"--f403043895ccc776c6055e8fae42\r\n" +
	"Content-Type: application/octet-stream; \r\n" +
	"	name=\"%s\"\r\n" +
	"Content-Disposition: attachment; \r\n" +
	"	filename=\"%s\"\r\n" +
	"Content-Transfer-Encoding: base64\r\n" +
	"X-Attachment-Id: 15fe33f078e2fcfdb251\r\n" +
	"Content-ID: <15fe33f078e2fcfdb251>\r\n" +
	"\r\n" +
	"%s\r\n" +
	"--f403043895ccc776c6055e8fae42--\r\n"

func sendToKindle(cmd *cobra.Command, args []string) {
	to := args[0]
	f := args[1]
	c, err := ioutil.ReadFile(f)
	if err != nil {
		log.Fatalf("reading %s: %s", f, err)
	}
	s := base64.StdEncoding.EncodeToString(c)
	b := path.Base(f)
	m := []byte(fmt.Sprintf(msgFmt, to, b, b, s))
	at := strings.LastIndex(to, "@")
	if at < 0 {
		log.Fatalf("address must contain @: %s", to)
	}
	mxs, err := net.LookupMX(to[at+1:])
	if err != nil {
		log.Fatal(err)
	}
	mx := fmt.Sprintf("%s:25", mxs[0].Host)
	conn, err := net.DialTimeout("tcp4", mx, 10*time.Second)
	if err != nil {
		log.Fatal(err)
	}
	defer func() { _ = conn.Close() }()
	client, err := smtp.NewClient(conn, "ceh.bz")
	if err != nil {
		log.Fatal(err)
	}
	client.Mail("ceh@ceh.bz")
	if err != nil {
		log.Fatal(err)
	}
	err = client.Rcpt(to)
	if err != nil {
		log.Fatal(err)
	}
	body, err := client.Data()
	if err != nil {
		log.Fatal(err)
	}
	n, err := body.Write(m)
	if err == nil && n < len(m) {
		err = io.ErrShortWrite
	}
	if err != nil {
		log.Fatal(err)
	}
	err = body.Close()
	if err != nil {
		log.Fatal(err)
	}
	err = client.Quit()
	// err = smtp.SendMail(mx, nil, "haynes@edgelay.org", []string{to}, m)
	if err != nil {
		log.Fatal(err)
	}
}

func init() {
	cobra.OnInitialize(initConfig)

	// Here you will define your flags and configuration settings.
	// Cobra supports persistent flags, which, if defined here,
	// will be global for your application.
	RootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.sendtokindle.yaml)")

	// Cobra also supports local flags, which will only run
	// when this action is called directly.
	RootCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	if cfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)
	} else {
		// Find home directory.
		home, err := homedir.Dir()
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		// Search config in home directory with name ".sendtokindle" (without extension).
		viper.AddConfigPath(home)
		viper.SetConfigName(".sendtokindle")
	}

	viper.AutomaticEnv() // read in environment variables that match

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err == nil {
		fmt.Println("Using config file:", viper.ConfigFileUsed())
	}
}
