// PeeTwoPee native capture helper. PCM exists only in bounded pipes/memory; never files.
using System;using System.IO;using System.Diagnostics;using System.Runtime.InteropServices;using System.Threading;
[ComImport,Guid("1CB9AD4C-DBFA-4C32-B178-C2F568A703B2"),InterfaceType(ComInterfaceType.InterfaceIsIUnknown)] interface Client {
 [PreserveSig]int Initialize(int share,uint flags,long duration,long period,ref Format format,IntPtr session);
 [PreserveSig]int GetBufferSize(out uint n);[PreserveSig]int GetStreamLatency(out long n);[PreserveSig]int GetCurrentPadding(out uint n);[PreserveSig]int IsFormatSupported(int mode,IntPtr f,IntPtr closest);[PreserveSig]int GetMixFormat(out IntPtr f);[PreserveSig]int GetDevicePeriod(out long a,out long b);
 [PreserveSig]int Start();[PreserveSig]int Stop();[PreserveSig]int Reset();[PreserveSig]int SetEventHandle(IntPtr h);[PreserveSig]int GetService(ref Guid id,[MarshalAs(UnmanagedType.IUnknown)]out object service);
}
[StructLayout(LayoutKind.Sequential,Pack=2)]struct Format{public ushort Tag,Channels;public uint Rate,BytesPerSecond;public ushort Align,Bits,Extra;}
[ComImport,Guid("C8ADBD64-E71E-48A0-A4DE-185C395CD317"),InterfaceType(ComInterfaceType.InterfaceIsIUnknown)]interface Capture{[PreserveSig]int GetBuffer(out IntPtr data,out uint frames,out uint flags,out ulong position,out ulong time);[PreserveSig]int ReleaseBuffer(uint n);[PreserveSig]int GetNextPacketSize(out uint n);}
[ComImport,Guid("72A22D78-CDE4-431D-B8CC-843A71199B6D"),InterfaceType(ComInterfaceType.InterfaceIsIUnknown)]public interface Operation{[PreserveSig]int GetActivateResult(out int result,[MarshalAs(UnmanagedType.IUnknown)]out object client);}
[Guid("41D949AB-9862-444A-80F6-C261334DA5EB"),InterfaceType(ComInterfaceType.InterfaceIsIUnknown)]public interface Completion{[PreserveSig]int ActivateCompleted(Operation op);}
[Guid("94EA2B94-E9CC-49E0-C0FF-EE64CA8F5B90"),InterfaceType(ComInterfaceType.InterfaceIsIUnknown)]public interface Agile{}
[ComVisible(true),ClassInterface(ClassInterfaceType.None)]public class Completed:Completion,Agile {public ManualResetEvent Ready=new ManualResetEvent(false);public object Audio;public int Result;public int ActivateCompleted(Operation op){try{int hr;object a;Result=op.GetActivateResult(out hr,out a);if(Result>=0){Result=hr;Audio=a;}}catch(Exception e){Result=Marshal.GetHRForException(e);}finally{Ready.Set();}return 0;}}
[StructLayout(LayoutKind.Explicit,Size=24)]struct Variant{[FieldOffset(0)]public ushort Type;[FieldOffset(8)]public int Size;[FieldOffset(16)]public IntPtr Data;}
[ComImport,Guid("A95664D2-9614-4F35-A746-DE8DB63617E6"),InterfaceType(ComInterfaceType.InterfaceIsIUnknown)]interface Enumerator{[PreserveSig]int EnumAudioEndpoints(int a,int b,out IntPtr devices);[PreserveSig]int GetDefaultAudioEndpoint(int flow,int role,out Device device);}
[ComImport,Guid("D666063F-1587-4E43-81F1-B948E807363F"),InterfaceType(ComInterfaceType.InterfaceIsIUnknown)]interface Device{[PreserveSig]int Activate(ref Guid id,uint context,IntPtr p,[MarshalAs(UnmanagedType.IUnknown)]out object result);}
class Program {
 [DllImport("Mmdevapi.dll",CharSet=CharSet.Unicode)]static extern int ActivateAudioInterfaceAsync(string path,ref Guid iid,ref Variant p,Completion handler,out Operation op);
 static void Check(int hr){if(hr<0)Marshal.ThrowExceptionForHR(hr);}
 [MTAThread]static int Main(string[] args){Client client=null;Capture capture=null;object enumerator=null,device=null;try{
 if(args.Length==1&&args[0]=="--list"){foreach(Process p in Process.GetProcesses()){try{if(p.MainWindowHandle!=IntPtr.Zero)Console.WriteLine(p.Id+"|"+p.ProcessName+"|"+Convert.ToBase64String(System.Text.Encoding.UTF8.GetBytes(p.MainWindowTitle)));}catch{}finally{p.Dispose();}}return 0;}
 if(args.Length!=3||args[0]!="--capture")throw new ArgumentException("Explicit capture mode and PID required");
 string mode=args[1];uint pid=uint.Parse(args[2]);Guid iid=new Guid("1CB9AD4C-DBFA-4C32-B178-C2F568A703B2");
 if(mode=="device"){
 enumerator=Activator.CreateInstance(Type.GetTypeFromCLSID(new Guid("BCDE0395-E52F-467C-8E3D-C4579291692E")));Device d;Check(((Enumerator)enumerator).GetDefaultAudioEndpoint(0,0,out d));device=d;object obj;Check(d.Activate(ref iid,23,IntPtr.Zero,out obj));client=(Client)obj;
 }else{
 if(mode!="process"&&mode!="exclude")throw new ArgumentException("Invalid mode");
 IntPtr data=Marshal.AllocHGlobal(12);Completed cb=new Completed();Operation op=null;
 try{Marshal.WriteInt32(data,0,1);Marshal.WriteInt32(data,4,unchecked((int)pid));Marshal.WriteInt32(data,8,mode=="exclude"?1:0);Variant v=new Variant{Type=65,Size=12,Data=data};Check(ActivateAudioInterfaceAsync("VAD\\Process_Loopback",ref iid,ref v,cb,out op));if(!cb.Ready.WaitOne(5000))throw new TimeoutException("Process loopback activation timed out");Check(cb.Result);client=(Client)cb.Audio;}finally{if(cb.Ready.WaitOne(0))Marshal.FreeHGlobal(data);if(op!=null)Marshal.ReleaseComObject(op);GC.KeepAlive(cb);}
 }
 Format f=new Format{Tag=1,Channels=1,Rate=48000,BytesPerSecond=96000,Align=2,Bits=16,Extra=0};
 // Shared loopback with Windows conversion to fixed 48k mono PCM, no exclusive device.
 Check(client.Initialize(0,0x80020000,1000000,0,ref f,IntPtr.Zero));object service;Guid captureId=new Guid("C8ADBD64-E71E-48A0-A4DE-185C395CD317");Check(client.GetService(ref captureId,out service));capture=(Capture)service;
 Stream output=Console.OpenStandardOutput();Check(client.Start());Console.Error.WriteLine("READY WASAPI 48000 mono pcm16 "+mode);byte[] buffer=new byte[192000];
 for(;;){uint n;Check(capture.GetNextPacketSize(out n));if(n==0){Thread.Sleep(5);continue;}IntPtr raw;uint flags;ulong pos,time;Check(capture.GetBuffer(out raw,out n,out flags,out pos,out time));try{int bytes=checked((int)n*2);if(bytes>buffer.Length)throw new InvalidOperationException("Capture buffer exceeded limit");if((flags&2)!=0)Array.Clear(buffer,0,bytes);else Marshal.Copy(raw,buffer,0,bytes);output.Write(buffer,0,bytes);}finally{Check(capture.ReleaseBuffer(n));}}
 }catch(Exception e){Console.Error.WriteLine("WASAPI_ERROR 0x"+Marshal.GetHRForException(e).ToString("X8")+" "+e.GetType().Name);return 1;}finally{if(client!=null){client.Stop();}if(capture!=null)Marshal.ReleaseComObject(capture);if(client!=null)Marshal.ReleaseComObject(client);if(device!=null)Marshal.ReleaseComObject(device);if(enumerator!=null)Marshal.ReleaseComObject(enumerator);}}
}
